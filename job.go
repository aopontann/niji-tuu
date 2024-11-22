package nsa

import (
	"context"
	"log/slog"
	"regexp"
	"slices"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/uptrace/bun"
)

type Job struct {
	db  *DB
	yt  *Youtube
	fcm *FCM
	task *Task
}

func NewJobs(youtubeApiKey string, db *bun.DB) *Job {
	return &Job{
		yt:  NewYoutube(youtubeApiKey),
		db:  NewDB(db),
		fcm: NewFCM(),
		task: NewTask(),
	}
}

func (j *Job) CheckNewVideoJob() error {
	pids, err := j.db.PlaylistIDs()
	if err != nil {
		slog.Error("playlists-ids",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	// 公開前、公開中の動画IDを取得
	upcomingLiveVideoIDs, err := j.yt.UpcomingLiveVideoIDs(pids)
	if err != nil {
		slog.Error("upcoming-live-video-ids",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	// RSSから過去30分間にアップロードされた動画IDを取得
	rssVideoIDs, err := j.yt.RssFeed(pids)
	if err != nil {
		slog.Error("rss-feed",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	// 重複している動画IDを削除する準備
	joinVIDs := append(upcomingLiveVideoIDs, rssVideoIDs...)
	slices.Sort(joinVIDs)

	// 動画情報を取得
	videos, err := j.yt.Videos(slices.Compact(joinVIDs))
	if err != nil {
		slog.Error("videos-list",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	// DBに登録されていない動画情報のみにフィルター
	notExistsVideos, err := j.db.NotExistsVideos(videos)
	if err != nil {
		slog.Error("not-exists-videos",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	// 歌ってみた動画ゲリラ対応
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
		if !j.yt.FindSongKeyword(v) || j.yt.FindIgnoreKeyword(v) {
			continue
		}
		// 5分以内に公開される動画
		sst, _ := time.Parse("2006-01-02T15:04:05Z", v.LiveStreamingDetails.ScheduledStartTime)
		sub := time.Now().UTC().Sub(sst).Minutes()
		if sub < 5 && sub >= 0 {
			tokens, err := j.db.getSongTokens()
			if err != nil {
				slog.Error("get-song-tokens",
					slog.String("severity", "ERROR"),
					slog.String("message", err.Error()),
				)
				return err
			}
			j.fcm.Notification(
				"まもなく公開",
				tokens,
				&NotificationVideo{
					ID:        v.Id,
					Title:     v.Snippet.Title,
					Thumbnail: v.Snippet.Thumbnails.High.Url,
				},
			)
		}
	}

	// 歌みた動画か判別しづらい動画をメールに送信する
	for _, v := range notExistsVideos {
		if j.yt.FindSongKeyword(v) {
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
		if j.yt.FindIgnoreKeyword(v) {
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
		if !j.yt.FindSongKeyword(v) || j.yt.FindIgnoreKeyword(v) {
			continue
		}
		err = j.task.CreateSongTask(v)
		// 検証中のため、エラーを返さない
		// if err != nil {
		// 	return err
		// }
	}

	// 3回までリトライ　1秒後にリトライ
	err = retry.Do(
		func() error {
			// 動画情報をDBに登録
			err = j.db.SaveVideos(videos)
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

func (j *Job) SongVideoAnnounceJob() error {
	// 5分後にプレミア公開される歌みた動画を取得
	videos, err := j.db.songVideos5m()
	if err != nil {
		slog.Error("song-videos-5m",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}
	if len(videos) == 0 {
		return nil
	}

	// FCMトークンを取得
	tokens, err := j.db.getSongTokens()
	if err != nil {
		slog.Error("get-song-tokens",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	for _, v := range videos {
		slog.Info("song-video-announce",
			slog.String("severity", "INFO"),
			slog.String("video_id", v.ID),
			slog.String("title", v.Title),
		)
		isExists, err := j.yt.IsExistsVideo(v.ID)
		if err != nil {
			slog.Error("is-exists-video",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
		}
		if !isExists {
			slog.Warn("is-exists-video",
				slog.String("severity", "WARNING"),
				slog.String("id", v.ID),
				slog.String("message", "deleted video"),
			)
			continue
		}

		err = NewMail().Subject("5分後に公開").Id(v.ID).Title(v.Title).Send()
		if err != nil {
			slog.Error("mail-send",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			return err
		}

		// push通知
		err = j.fcm.Notification(
			"5分後に公開",
			tokens,
			&NotificationVideo{
				ID:        v.ID,
				Title:     v.Title,
				Thumbnail: v.Thumbnail,
			})
		if err != nil {
			slog.Error("notification",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			return err
		}
	}
	return nil
}

// キーワード告知
func (j *Job) KeywordAnnounceJob() error {
	ctx := context.Background()
	now, _ := time.Parse(time.RFC3339, time.Now().UTC().Format("2006-01-02T15:04:00Z"))
	tAfter := now.Add(-20 * time.Minute)
	tBefore := now.Add(-10 * time.Minute)
	var videos []Video
	err := j.db.Service.NewSelect().Model(&videos).Where("? BETWEEN ? AND ?", bun.Ident("created_at"), tAfter, tBefore).Scan(ctx)
	if err != nil {
		return err
	}
	if len(videos) == 0 {
		return nil
	}

	topics, err := j.db.getTopicsUserRegister()
	if err != nil {
		slog.Error("getTopicsUserRegister",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	for _, topic := range topics {
		reg := ".*" + topic.Name + ".*"
		for _, v := range videos {
			// キーワードに一致した場合
			if regexp.MustCompile(reg).Match([]byte(v.Title)) {
				slog.Info("TopicNotification",
					slog.String("severity", "INFO"),
					slog.String("topic_name", topic.Name),
					slog.String("video_id", v.ID),
					slog.String("video_title", v.Title),
				)
				err := j.fcm.TopicNotification(topic.Name, &NotificationVideo{
					ID:        v.ID,
					Title:     v.Title,
					Thumbnail: v.Thumbnail,
				})
				if err != nil {
					slog.Error("TopicNotification",
						slog.String("severity", "ERROR"),
						slog.String("message", err.Error()),
					)
					return err
				}
			}
		}
	}
	return nil
}
