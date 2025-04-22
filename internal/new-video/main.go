package newvideo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/aopontann/niji-tuu/internal/common/db"
	"github.com/aopontann/niji-tuu/internal/common/youtube"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-retryablehttp"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	err := CheckNewVideoJob()
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CheckNewVideoJob() error {
	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	cdb, err := db.NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}
	defer cdb.Close()

	vtubers, err := GetStatusChengedVtubers(yt, cdb)
	if err != nil {
		return err
	}

	// playlistで動画数を取得しても、PlaylistItemsに反映されるまでに時間がかかる？
	// 反映されるのに時間が必要そうだから、10秒待つ処理入れる
	time.Sleep(10 * time.Second)

	vids, err := GetNewVideoIDs(yt, cdb, vtubers)
	if err != nil {
		return err
	}

	rssVIDs, err := GetNewVideoIDsWithRSS(yt, cdb)
	if err != nil {
		return err
	}

	// RSSのみで取得した動画IDを表示（RSSが必要か確認するために一時的に表示）
	for _, rvid := range rssVIDs {
		if !slices.Contains(vids, rvid) {
			slog.Info("RSSのみで取得できた動画がありました")
		}
	}

	// 動画IDリストを結合して重複削除処理をする
	joinedVIDs := append(vids, rssVIDs...)
	slices.Sort(joinedVIDs)
	vids = slices.Compact(joinedVIDs)

	// 新着動画がない場合、処理を終了
	if len(vids) == 0 {
		// DBのプレイリスト動画数を更新
		err = cdb.UpdateVtubers(vtubers, nil)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
	}

	// メン限、限定公開で動画情報を取得できない場合があるため、先に動画IDのみをログ表示
	slog.Info("new-video-ids",
		slog.String("video_id", strings.Join(vids, ",")),
	)

	// ログ表示のため動画情報を取得
	videos, err := yt.Videos(vids)
	if err != nil {
		return err
	}

	// メン限定、限定公開の動画があった場合
	if len(vids) != len(videos) {
		slog.Warn("メン限、限定公開の動画が含まれています")
	}

	// メン限、限定公開の動画情報はAPIの仕様上取得できない
	// 新着動画の検知はしているが、1つも動画が取得できなかった場合、プレイリスト情報を更新して処理を終了
	if len(videos) == 0 {
		// DBのプレイリスト動画数を更新
		err = cdb.UpdateVtubers(vtubers, nil)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
		return nil
	}

	// 確認用ログ
	for _, v := range videos {
		slog.Info("new-videos",
			slog.String("video_id", v.Id),
			slog.String("title", v.Snippet.Title),
		)
	}

	// トランザクション開始
	ctx := context.Background()
	tx, err := cdb.Service.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	// DBのプレイリスト動画数を更新
	err = cdb.UpdateVtubers(vtubers, &tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	// 動画情報をDBに登録
	err = cdb.SaveVideos(videos, &tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	// コミット
	err = tx.Commit()
	if err != nil {
		return err
	}

	err = NewVideoWebHook(vids)
	return err
}

// 動画数もしくはプレイリストのURLが変更されたvtuber情報を取得
// vtuber情報はYouube Data APIから取得した最新の状態が格納されている
func GetStatusChengedVtubers(yt *youtube.Youtube, cdb *db.DB) ([]db.Vtuber, error) {
	// DBに登録されているプレイリストの動画数を取得
	vtuber, err := cdb.GetVtubers()
	if err != nil {
		return nil, err
	}

	// Youtube Data API から最新のプレイリストの動画数を取得
	var pids []string
	for _, vt := range vtuber {
		pid := strings.Replace(vt.ID, "UC", "UU", 1)
		pids = append(pids, pid)
	}
	newPlaylists, err := yt.Playlists(pids)
	if err != nil {
		return nil, err
	}

	var chengedVtuber []db.Vtuber
	for _, vt := range vtuber {
		pid := strings.Replace(vt.ID, "UC", "UU", 1)
		if vt.ItemCount != newPlaylists[pid].ItemCount || vt.PlaylistLatestUrl != newPlaylists[pid].Url {
			chengedVtuber = append(chengedVtuber, db.Vtuber{
				ID:                vt.ID,
				Name:              vt.Name,
				ItemCount:         newPlaylists[pid].ItemCount,
				PlaylistLatestUrl: newPlaylists[pid].Url,
				CreatedAt:         vt.CreatedAt,
				UpdatedAt:         time.Now(),
			})

			slog.Info("changedPlaylist",
				slog.String("playlist_id", pid),
				slog.Int64("old_item_count", vt.ItemCount),
				slog.Int64("new_item_count", newPlaylists[pid].ItemCount),
				slog.String("old_url", vt.PlaylistLatestUrl),
				slog.String("new_url", newPlaylists[pid].Url),
			)

			if newPlaylists[pid].ItemCount < vt.ItemCount {
				slog.Warn("動画が削除されている可能性があります")
			}
		}
	}

	return chengedVtuber, nil
}

// 新しくアップロードされた動画IDを取得
func GetNewVideoIDs(yt *youtube.Youtube, cdb *db.DB, vtubers []db.Vtuber) ([]string, error) {
	var pids []string
	for _, vt := range vtubers {
		pid := strings.Replace(vt.ID, "UC", "UU", 1)
		pids = append(pids, pid)
	}

	// 更新されたプレイリストの動画IDリストを取得
	vids, err := yt.PlaylistItems(pids)
	if err != nil {
		return nil, err
	}

	// DBに登録されていない動画のみにフィルター
	newVIDs, err := cdb.NotExistsVideoID(vids)
	if err != nil {
		return nil, err
	}

	return newVIDs, nil
}

// RSSから全てのチャンネルから新しくアップロードされた動画IDを取得
func GetNewVideoIDsWithRSS(yt *youtube.Youtube, cdb *db.DB) ([]string, error) {
	pids, err := cdb.PlaylistIDs()
	if err != nil {
		return nil, err
	}

	// RSS から新着動画IDを取得
	vids, err := yt.RssFeed(pids)
	if err != nil {
		return nil, err
	}

	// DBに登録されていない動画のみにフィルター
	newVIDs, err := cdb.NotExistsVideoID(vids)
	if err != nil {
		return nil, err
	}

	return newVIDs, nil
}

// 新しい動画がアップロードされた動画IDを含めたHTTPリクエストを送信
func NewVideoWebHook(vids []string) error {
	if len(vids) == 0 {
		return nil
	}
	// 登録されたURLにリクエストを送信
	// リクエストが失敗してもリトライするように
	// どれかのリクエストが失敗しても、他のリクエストには影響が出ないように
	vidsStr := strings.Join(vids, ",")
	callback_urls := []string{
		os.Getenv("SONG_TASK_URL"),
		os.Getenv("DISCORD_TASK_URL"),
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 2

	var meg multierror.Group

	for _, url := range callback_urls {
		customURL := fmt.Sprintf("%s?v=%s", url, vidsStr)
		meg.Go(func() error {
			resp, err := retryClient.Post(customURL, "application/json", nil)
			resp.Body.Close()
			if err != nil {
				slog.Error(err.Error())
				return err
			}
			return nil
		})
	}

	merr := meg.Wait()
	return merr.ErrorOrNil()
}
