package nsa

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-retryablehttp"
	"google.golang.org/api/youtube/v3"
)

func CheckNewVideoJob() error {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}
	defer db.Close()

	// DBに登録されているプレイリストの動画数を取得
	oldPlaylists, err := db.Playlists()
	if err != nil {
		return err
	}

	// Youtube Data API から最新のプレイリストの動画数を取得
	var pids []string
	for pid := range oldPlaylists {
		pids = append(pids, pid)
	}
	newPlaylists, err := yt.Playlists(pids)
	if err != nil {
		return err
	}

	// playlistで動画数を取得しても、PlaylistItemsに反映されるまでに時間がかかる？
	// 反映されるのに時間が必要そうだから、10秒待つ処理入れる
	time.Sleep(10 * time.Second)

	// 新しくアップロードされた動画ID
	newVIDs, err := GetNewVideoIDs(yt, db, oldPlaylists, newPlaylists)
	if err != nil {
		return err
	}

	// Playlist更新用にデータを整形
	changedPlaylist := make(map[string]Playlist, 500)
	for pid, playlist := range newPlaylists {
		if playlist.ItemCount != oldPlaylists[pid].ItemCount || playlist.Url != oldPlaylists[pid].Url {
			changedPlaylist[pid] = Playlist{ItemCount: playlist.ItemCount, Url: playlist.Url}
		}
	}

	if len(newVIDs) == 0 {
		// DBのプレイリスト動画数を更新
		err = retry.Do(
			func() error {
				return db.UpdatePlaylistItem(changedPlaylist)
			},
			retry.Attempts(3),
			retry.Delay(1*time.Second),
		)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
		return nil
	}

	// ログ表示のため動画情報を取得
	videos, err := yt.Videos(newVIDs)
	if err != nil {
		return err
	}

	// 確認用ログ
	for _, v := range videos {
		slog.Info("new-videos",
			slog.String("video_id", v.Id),
			slog.String("title", v.Snippet.Title),
		)
	}

	// 3回までリトライ　1秒後にリトライ
	err = retry.Do(
		func() error {
			// トランザクション開始
			ctx := context.Background()
			tx, err := db.Service.BeginTx(ctx, &sql.TxOptions{})
			if err != nil {
				return err
			}

			// DBのプレイリスト動画数を更新
			err = db.UpdatePlaylistItemWithTx(tx, changedPlaylist)
			if err != nil {
				tx.Rollback()
				return err
			}
			// 動画情報をDBに登録
			err = db.SaveVideos(tx, videos)
			if err != nil {
				return err
			}

			err = tx.Commit()
			if err != nil {
				return err
			}
			return nil
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
	)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	err = NewVideoWebHook(newVIDs)
	return err
}

// 新しい動画がアップロードされた動画IDを含めたHTTPリクエストを送信
func NewVideoWebHook(vids []string) error {
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

// 受け取った動画IDが歌動画か解析し、歌動画だった場合はタスクを登録する
func SongVideoCheck(vids []string) error {
	slog.Info("処理開始",
		slog.String("vids", strings.Join(vids, ",")),
	)
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	task, err := NewTask()
	if err != nil {
		return err
	}

	videos, err := yt.Videos(vids)
	if err != nil {
		return err
	}

	// エラー時にgoroutineの中断をしてほしくないのと、
	// 発生したエラーを全てハンドリングしたいので go-multierror を採用
	var meg multierror.Group

	meg.Go(func() error {
		err = retry.Do(
			func() error {
				return SendMailMaybeSongVideos(yt, videos)
			},
			retry.Attempts(3),
			retry.Delay(1*time.Second),
		)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
		return nil
	})

	meg.Go(func() error {
		err = retry.Do(
			func() error {
				return AddSongTaskToCloudTasks(yt, task, videos)
			},
			retry.Attempts(3),
			retry.Delay(1*time.Second),
		)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
		return nil
	})

	merr := meg.Wait()
	slog.Info("処理終了")
	return merr.ErrorOrNil()
}

// 歌動画通知
func SongVideoAnnounceJob(vid string) error {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}
	defer db.Close()
	fcm := NewFCM()

	// 動画か消されていないかチェック
	videos, err := yt.Videos([]string{vid})
	if err != nil {
		return err
	}
	if len(videos) == 0 {
		slog.Warn("deleted video",
			slog.String("video_id", vid),
		)
		return nil
	}

	// FCMトークンを取得
	tokens, err := db.getSongTokens()
	if err != nil {
		return err
	}

	title := videos[0].Snippet.Title
	thumbnail := videos[0].Snippet.Thumbnails.High.Url

	slog.Info("song-video-announce",
		slog.String("video_id", vid),
		slog.String("title", title),
	)

	// 動作確認用としてメールを送信
	err = NewMail().Subject("5分後に公開").Id(vid).Title(title).Send()
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	err = fcm.Notification(
		"5分後に公開",
		tokens,
		&NotificationVideo{
			ID:        vid,
			Title:     title,
			Thumbnail: thumbnail,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// 受け取った動画IDから動画の公開予定時刻を取得し、公開1時間前に通知するタスクを登録
func CreateTaskToNoficationByDiscord(vids []string) error {
	slog.Info("処理開始",
		slog.String("vids", strings.Join(vids, ",")),
	)
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	task, err := NewTask()
	if err != nil {
		return err
	}

	videos, err := yt.Videos(vids)
	if err != nil {
		return err
	}

	// discord から通知するタスクを登録
	for _, v := range videos {
		err = task.Create(&TaskInfo{
			Video: v,
			QueueID: os.Getenv("DISCORD_QUEUE_ID"),
			URL: os.Getenv("DISCORD_URL"),
			MinutesAgo: time.Hour,
		})
		if err != nil {
			return err
		}
	}
	slog.Info("処理終了")
	return nil
}

// discordに通知
func DiscordAnnounceJob(vid string) error {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}
	defer db.Close()

	// 動画か消されていないかチェック
	videos, err := yt.Videos([]string{vid})
	if err != nil {
		return err
	}
	if len(videos) == 0 {
		slog.Warn("deleted video",
			slog.String("video_id", vid),
		)
		return nil
	}

	title := videos[0].Snippet.Title

	slog.Info("discord-announce",
		slog.String("video_id", vid),
		slog.String("title", title),
	)

	roles, err := db.GetRoles()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		slog.Error(err.Error())
		return err
	}

	for _, role := range roles {
		// 小文字に統一してから一致チェック
		titleLower := strings.ToLower(title)

		// キーワードに一致するか
		keywords := strings.Join(role.Keywords, "|")
		keywordsLower := strings.ToLower(keywords)
		regPattern := ".*" + keywordsLower + ".*"
		regex, _ := regexp.Compile(regPattern)
		if !regex.MatchString(titleLower) {
			continue
		}

		// 除外するキーワードに一致した場合
		exclusionkeywords := strings.Join(role.ExclusionKeywords, "|")
		exclusionKeywordsLower := strings.ToLower(exclusionkeywords)
		regPattern = ".*" + exclusionKeywordsLower + ".*"
		regex, _ = regexp.Compile(regPattern)
		if len(role.ExclusionKeywords) != 0 && regex.MatchString(titleLower) {
			fmt.Println("除外するキーワードに一致するため、除外されました")
			continue
		}

		// キーワードに一致した場合
		body := []byte(fmt.Sprintf(`{"content": "<@&%s>\nhttps://www.youtube.com/watch?v=%s"}`, role.ID, vid))
		resp, err := http.Post(
			role.WebhookURL,
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}

	return nil
}

// 歌みた動画か判別しづらい動画をメールに送信する
func SendMailMaybeSongVideos(yt *Youtube, videos []youtube.Video) error {
	for _, v := range videos {
		if yt.FindSongKeyword(v) {
			continue
		}
		if v.LiveStreamingDetails == nil {
			continue
		}
		if v.Snippet.LiveBroadcastContent != "upcoming" {
			continue
		}
		if v.ContentDetails.Duration == "P0D" {
			continue
		}
		// 特定のキーワードを含んでいる場合
		if yt.FindIgnoreKeyword(v) {
			continue
		}

		err := NewMail().Subject("歌みた動画判定").Id(v.Id).Title(v.Snippet.Title).Send()
		if err != nil {
			slog.Error(err.Error())
			return err
		}
	}
	return nil
}

// cloud task に歌みた告知タスクを登録
func AddSongTaskToCloudTasks(yt *Youtube, task *Task, videos []youtube.Video) error {
	for _, v := range videos {
		// 生放送ではない、プレミア公開されない動画の場合
		if v.LiveStreamingDetails == nil {
			continue
		}
		// 放送終了した場合
		if v.Snippet.LiveBroadcastContent == "none" {
			continue
		}
		// 生放送の場合
		if v.ContentDetails.Duration == "P0D" {
			continue
		}
		if !yt.FindSongKeyword(v) || yt.FindIgnoreKeyword(v) {
			continue
		}
		err := task.Create(&TaskInfo{
			Video: v,
			QueueID: os.Getenv("SONG_QUEUE_ID"),
			URL: os.Getenv("SONG_URL"),
			MinutesAgo: time.Minute * 5,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// YouTube Data API から新着動画IDを取得
func GetNewVideoIDs(yt *Youtube, db *DB, oldPlaylists map[string]Playlist, newPlaylists map[string]Playlist) ([]string, error) {
	var ytVIDs []string
	for pid, playlist := range newPlaylists {
		if playlist.ItemCount == oldPlaylists[pid].ItemCount && playlist.Url == oldPlaylists[pid].Url {
			continue
		}

		slog.Info("changedPlaylist",
			slog.String("playlist_id", pid),
			slog.Int64("old_item_count", oldPlaylists[pid].ItemCount),
			slog.Int64("new_item_count", playlist.ItemCount),
			slog.String("old_url", oldPlaylists[pid].Url),
			slog.String("new_url", playlist.Url),
		)

		if playlist.ItemCount < oldPlaylists[pid].ItemCount {
			// 動画が削除されても、新しい動画がアップロードされている可能性もあるためcontinueしない
			slog.Warn("動画が削除されている可能性があります")
		}

		// 更新されたプレイリストの動画IDリストを取得
		vids, err := yt.PlaylistItems(pid)
		if err != nil {
			return nil, err
		}

		ytVIDs = append(ytVIDs, vids...)
	}

	// RSS から新着動画IDを取得
	var pids []string
	for pid := range oldPlaylists {
		pids = append(pids, pid)
	}

	rssVIDs, err := yt.RssFeed(pids)
	if err != nil {
		return nil, err
	}

	// 動画IDリストを結合して重複削除処理をする
	joinedVIDs := append(ytVIDs, rssVIDs...)
	slices.Sort(joinedVIDs)
	vids := slices.Compact(joinedVIDs)

	// DBに登録されていない動画のみにフィルター
	newVIDs, err := db.NotExistsVideoID(vids)
	if err != nil {
		return nil, err
	}

	return newVIDs, nil
}
