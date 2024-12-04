package nsa

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
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
	task, err := NewTask()
	if err != nil {
		return err
	}

	// DBに登録されているプレイリストの動画数を取得
	oldPlaylists, err := db.Playlists()
	if err != nil {
		return err
	}

	// プレイリストIDリスト
	var plist []string
	for pid := range oldPlaylists {
		plist = append(plist, pid)
	}

	// Youtube Data API から最新のプレイリストの動画数を取得
	newPlaylists, err := yt.Playlists(plist)
	if err != nil {
		return err
	}

	// 動画数が変化しているプレイリストIDを取得
	var changedPlaylistID []string
	for pid, itemCount := range oldPlaylists {
		if itemCount != newPlaylists[pid] {
			changedPlaylistID = append(changedPlaylistID, pid)
		}
	}

	// 新しくアップロードされた動画IDを取得
	newVIDs, err := yt.PlaylistItems(changedPlaylistID)
	if err != nil {
		return err
	}

	// 動画情報を取得
	videos, err := yt.Videos(newVIDs)
	if err != nil {
		return err
	}

	// 確認用ログ
	for _, v := range videos {
		slog.Info("videos",
			slog.String("severity", "INFO"),
			slog.String("video_id", v.Id),
			slog.String("title", v.Snippet.Title),
		)
	}

	// DBに登録されていない動画情報のみにフィルター
	notExistsVideos, err := db.NotExistsVideos(videos)
	if err != nil {
		return err
	}

	// 確認用ログ
	for _, v := range notExistsVideos {
		slog.Info("notExistsVideos",
			slog.String("severity", "INFO"),
			slog.String("video_id", v.Id),
			slog.String("title", v.Snippet.Title),
		)
	}

	// 歌みた動画か判別しづらい動画をメールに送信する
	for _, v := range notExistsVideos {
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
			slog.Error("mail-send",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			return err
		}
	}

	// cloud task に歌みた告知タスクを登録
	for _, v := range notExistsVideos {
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
		err = task.CreateSongTask(v)
		if err != nil {
			return err
		}
	}

	// Topic全件取得
	// 実際のプッシュ通知はユーザに登録されているTopicのみ通知する
	topics, err := db.getAllTopics()
	if err != nil {
		return err
	}

	for _, topic := range topics {
		regPattern := ".*" + topic.Name + ".*"
		regex, _ := regexp.Compile(regPattern)
		for _, v := range notExistsVideos {
			// キーワードに一致した場合
			if regex.MatchString(v.Snippet.Title) {
				err := task.CreateTopicTask(v, topic)
				if err != nil {
					return err
				}
			}
		}
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
			err = db.UpdatePlaylistItem(tx, newPlaylists)
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
		slog.Error("save-videos-retry",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	return nil
}

func CheckNewVideoJobWithRSS() error {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}
	defer db.Close()
	task, err := NewTask()
	if err != nil {
		return err
	}

	pids, err := db.PlaylistIDs()
	if err != nil {
		return err
	}

	// RSSから過去30分間にアップロードされた動画IDを取得
	rssVideoIDs, err := yt.RssFeed(pids)
	if err != nil {
		return err
	}

	// RSS検証
	notExistRssVIDs, err := db.NotExistsVideoID(rssVideoIDs)
	if err == nil {
		slog.Info("RssFeed",
			slog.String("severity", "INFO"),
			slog.String("notExistRssVIDs", strings.Join(notExistRssVIDs, ",")),
		)
	}

	if len(notExistRssVIDs) == 0 {
		return nil
	}

	// 確認用ログ
	slog.Info("notExistsVideoID",
		slog.String("severity", "INFO"),
		slog.String("notExistRssVIDs", strings.Join(notExistRssVIDs, ",")),
	)

	// 動画情報を取得
	videos, err := yt.Videos(notExistRssVIDs)
	if err != nil {
		return err
	}

	// 確認用ログ
	for _, v := range videos {
		slog.Info("notExistsVideos",
			slog.String("severity", "INFO"),
			slog.String("video_id", v.Id),
			slog.String("title", v.Snippet.Title),
		)
	}

	// cloud task に歌みた告知タスクを登録
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
		err = task.CreateSongTask(v)
		if err != nil {
			return err
		}
	}

	// Topic全件取得
	// 実際のプッシュ通知はユーザに登録されているTopicのみ通知する
	topics, err := db.getAllTopics()
	if err != nil {
		return err
	}

	for _, topic := range topics {
		regPattern := ".*" + topic.Name + ".*"
		regex, _ := regexp.Compile(regPattern)
		for _, v := range videos {
			// キーワードに一致した場合
			if regex.MatchString(v.Snippet.Title) {
				err := task.CreateTopicTask(v, topic)
				if err != nil {
					return err
				}
			}
		}
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
		slog.Error("save-videos-retry",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	return nil
}

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
		slog.Warn("Videos",
			slog.String("severity", "WARNING"),
			slog.String("video_id", vid),
			slog.String("message", "deleted video"),
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
		slog.String("severity", "INFO"),
		slog.String("video_id", vid),
		slog.String("title", title),
	)

	// 動作確認用としてメールを送信
	err = NewMail().Subject("5分後に公開").Id(vid).Title(title).Send()
	if err != nil {
		slog.Error("mail-send",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	// プッシュ通知
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

// キーワード告知
func TopicAnnounceJob(vid string, tid int) error {
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
		slog.Warn("Videos",
			slog.String("severity", "WARNING"),
			slog.String("video_id", vid),
			slog.String("message", "deleted video"),
		)
		return nil
	}

	title := videos[0].Snippet.Title
	thumbnail := videos[0].Snippet.Thumbnails.High.Url

	// 一人以上のユーザが登録しているTopicのみを取得
	// ユーザ誰一人も登録していないTopicはプッシュ通知を送らない
	topic, err := db.getTopicWhereUserRegister(tid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		slog.Error("getTopicWhereUserRegister",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	err = fcm.TopicNotification(topic.Name, &NotificationVideo{
		ID:        vid,
		Title:     title,
		Thumbnail: thumbnail,
	})
	if err != nil {
		return err
	}

	return nil
}
