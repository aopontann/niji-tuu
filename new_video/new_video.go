package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/aopontann/niji-tuu/internal"
	"github.com/avast/retry-go/v4"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/jmoiron/sqlx"
	"google.golang.org/api/youtube/v3"
)

var db *sqlx.DB

func main() {
	db = internal.NewDB(os.Getenv("DSN"))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		err := CheckNewVideoJob()
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		panic(err)
	}
}

func CheckNewVideoJob() error {
	// 新着動画がアップロードの検知方法は以下
	// DBに登録されているライバーの情報とYouTube Data APIから取得した情報を比較し
	// ・　アップロード済みの動画の数
	// ・　アップロードされた動画のプレイリストのサムネイル（プレイリスト 例：https://www.youtube.com/playlist?list=UU-6rZgmxZSIbq786j3RD5ow）
	// が一致しないライバー情報を取得
	// そこからライバーごとにアップロード済みの動画の最新順で取得する
	// 上記の方法でも、検知漏れが発生する可能性があるため、RSS、他の取得方法(検討中)でも検知するようにする
	vtubers, err := GetStatusChangedVtubers()
	if err != nil {
		return err
	}

	// playlistで動画数を取得しても、PlaylistItemsに反映されるラグがある？
	// 反映されるのに時間が必要そうだから、10秒待つ処理入れる
	time.Sleep(10 * time.Second)

	// 新しくアップロードされた動画IDを取得
	vids, err := GetNewVideoIDs(vtubers)
	if err != nil {
		return err
	}

	rssVIDs, err := GetNewVideoIDsWithRSS()
	if err != nil {
		return err
	}

	// RSSのみで取得した動画IDを表示（RSSが必要か確認するために一時的に表示）
	for _, rvid := range rssVIDs {
		if !slices.Contains(vids, rvid) {
			slog.Info("RSSのみで取得できた動画がありました",
				slog.String("vid", rvid))
		}
	}

	// 動画IDリストを結合して重複削除処理をする
	joinedVIDs := append(vids, rssVIDs...)
	slices.Sort(joinedVIDs)
	vids = slices.Compact(joinedVIDs)

	// トランザクション開始
	tx := db.MustBegin()

	// 新着動画がない場合、処理を終了
	if len(vids) == 0 {
		// DBのプレイリスト動画数を更新
		err := UpdateVtubers(vtubers, tx)
		if err != nil {
			slog.Error(err.Error())
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				slog.Error(err.Error())
			}
			return err
		}
		if commitErr := tx.Commit(); commitErr != nil {
			slog.Error(commitErr.Error())
			return commitErr
		}
		return nil
	}

	// メン限、限定公開で動画情報を取得できない場合があるため、先に動画IDのみをログ表示
	slog.Info("new-video-ids",
		slog.String("video_id", strings.Join(vids, ",")),
	)

	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
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
		err := UpdateVtubers(vtubers, tx)
		if err != nil {
			slog.Error(err.Error())
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				slog.Error(err.Error())
			}
			return err
		}
		if commitErr := tx.Commit(); commitErr != nil {
			slog.Error(commitErr.Error())
			return commitErr
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

	// DBのプレイリスト動画数を更新
	err = UpdateVtubers(vtubers, tx)
	if err != nil {
		slog.Error(err.Error())
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error(err.Error())
		}
		return err
	}
	// 動画情報をDBに登録
	err = SaveVideos(videos, tx)
	if err != nil {
		slog.Error(err.Error())
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error(err.Error())
		}
		return err
	}

	// コミット
	if commitErr := tx.Commit(); commitErr != nil {
		slog.Error(commitErr.Error())
		return commitErr
	}

	err = NewVideoWebHook(vids)
	return err
}

// GetStatusChangedVtubers 動画数もしくはプレイリストのURLが変更されたvtuber情報を取得
// vtuber情報はYouTube Data APIから取得した最新の状態が格納されている
func GetStatusChangedVtubers() ([]internal.Vtuber, error) {
	// DBに登録されているプレイリストの動画数を取得
	var vtubers []internal.Vtuber
	err := db.Select(&vtubers, "SELECT * FROM vtubers")
	if err != nil {
		return nil, err
	}

	// Youtube Data API から最新のプレイリストの動画数を取得
	var pids []string
	for _, vt := range vtubers {
		pid := strings.Replace(vt.ID, "UC", "UU", 1)
		pids = append(pids, pid)
	}

	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return nil, err
	}
	newPlaylists, err := yt.Playlists(pids)
	if err != nil {
		return nil, err
	}

	var changedVtuber []internal.Vtuber
	for _, vt := range vtubers {
		pid := strings.Replace(vt.ID, "UC", "UU", 1)
		if vt.ItemCount != newPlaylists[pid].ItemCount || vt.PlaylistLatestUrl != newPlaylists[pid].Url {
			changedVtuber = append(changedVtuber, internal.Vtuber{
				ID:                vt.ID,
				Name:              vt.Name,
				ItemCount:         newPlaylists[pid].ItemCount,
				PlaylistLatestUrl: newPlaylists[pid].Url})

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

	return changedVtuber, nil
}

// GetNewVideoIDs 新しくアップロードされた動画IDを取得
func GetNewVideoIDs(vtubers []internal.Vtuber) ([]string, error) {
	if len(vtubers) == 0 {
		return []string{}, nil
	}
	var pids []string
	for _, vt := range vtubers {
		pid := strings.Replace(vt.ID, "UC", "UU", 1)
		pids = append(pids, pid)
	}

	// 更新されたプレイリストの動画IDリストを取得
	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return nil, err
	}
	vids, err := yt.PlaylistItems(pids)
	if err != nil {
		return nil, err
	}

	notExistsID, err := FilterNotExistsVideoIDs(vids)
	if err != nil {
		return nil, err
	}

	return notExistsID, err
}

// GetNewVideoIDsWithRSS RSSから全てのチャンネルから新しくアップロードされた動画IDを取得
func GetNewVideoIDsWithRSS() ([]string, error) {
	var pids []string
	err := db.Select(&pids, "SELECT INSERT(id, 1, 2, 'UU') AS id FROM vtubers")
	if err != nil {
		return nil, err
	}

	// RSS から新着動画IDを取得
	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return nil, err
	}
	vids, err := yt.RssFeed(pids)
	if err != nil {
		return nil, err
	}

	notExistsID, err := FilterNotExistsVideoIDs(vids)
	if err != nil {
		return nil, err
	}

	return notExistsID, err
}

func FilterNotExistsVideoIDs(vids []string) ([]string, error) {
	if len(vids) == 0 {
		return []string{}, nil
	}

	query, args, err := sqlx.In("SELECT id FROM videos WHERE id IN (?)", vids)
	if err != nil {
		return nil, err
	}
	query = db.Rebind(query)

	var existsID []string
	err = db.Select(&existsID, query, args...)
	if err != nil {
		return nil, err
	}

	// 存在していない動画IDリスト
	var notExistsID []string
	for _, vid := range vids {
		if !slices.Contains(existsID, vid) {
			notExistsID = append(notExistsID, vid)
		}
	}

	return notExistsID, nil
}

func UpdateVtubers(vtubers []internal.Vtuber, tx *sqlx.Tx) error {
	if len(vtubers) == 0 {
		return nil
	}
	stmt, err := tx.PrepareNamed(internal.UpdateVtubersQuery)
	if err != nil {
		return err
	}
	for _, v := range vtubers {
		_, err := stmt.Exec(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func SaveVideos(videos []youtube.Video, tx *sqlx.Tx) error {
	var Videos []internal.Video
	for _, v := range videos {
		scheduledStartTime := "1998-01-01 15:04:05" // 例 2022-03-28T11:00:00Z
		if v.LiveStreamingDetails != nil {
			// "2022-03-28 11:00:00"形式に変換
			rep1 := strings.Replace(v.LiveStreamingDetails.ScheduledStartTime, "T", " ", 1)
			scheduledStartTime = strings.Replace(rep1, "Z", "", 1)
		}
		t, _ := time.Parse("2006-01-02 15:04:05", scheduledStartTime)
		Videos = append(Videos, internal.Video{
			ID:                 v.Id,
			Title:              v.Snippet.Title,
			Duration:           v.ContentDetails.Duration,
			Content:            v.Snippet.LiveBroadcastContent,
			ScheduledStartTime: t,
			UpdatedAt:          time.Now(),
		})
	}

	if len(Videos) == 0 {
		return nil
	}

	return retry.Do(
		func() error {
			_, err := tx.NamedExec(internal.InsertVideosQuery, Videos)
			return err
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
	)
}

// NewVideoWebHook 新しい動画がアップロードされた動画IDを含めたHTTPリクエストを送信
func NewVideoWebHook(vids []string) error {
	if len(vids) == 0 {
		return nil
	}
	// 登録されたURLにリクエストを送信
	// リクエストが失敗してもリトライするように
	// どれかのリクエストが失敗しても、他のリクエストには影響が出ないように
	vidsStr := strings.Join(vids, ",")
	callbackUrls := []string{
		os.Getenv("SONG_TASK_URL"),
		os.Getenv("DISCORD_TASK_URL"),
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 2

	var meg multierror.Group

	for _, url := range callbackUrls {
		customURL := fmt.Sprintf("%s?v=%s", url, vidsStr)
		meg.Go(func() error {
			resp, err := retryClient.Post(customURL, "application/json", nil)
			if err != nil {
				slog.Error(err.Error())
				return err
			}
			defer resp.Body.Close()
			// keepAliveできずにコネクションが再利用されずに終了してしまうため、bodyを読みきる
			// io.Discardは書き込んだバイトを全て捨てる
			_, err = io.Copy(io.Discard, resp.Body)
			if err != nil {
				return err
			}
			return nil
		})
	}

	merr := meg.Wait()
	return merr.ErrorOrNil()
}
