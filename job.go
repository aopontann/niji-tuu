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
	task, err := NewTask()
	if err != nil {
		return err
	}

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

	// 動画情報を取得
	videos, err := yt.Videos(newVIDs)
	if err != nil {
		return err
	}

	// 確認用ログ
	for _, v := range videos {
		slog.Info("videos",
			slog.String("video_id", v.Id),
			slog.String("title", v.Snippet.Title),
		)
	}

	if os.Getenv("ENV") == "prod" {
		// 歌みた動画か判別しづらい動画をメールに送信する
		err = SendMailMaybeSongVideos(yt, videos)
		if err != nil {
			return err
		}

		// cloud task に歌みた告知タスクを登録
		err = AddSongTaskToCloudTasks(yt, task, videos)
		if err != nil {
			return err
		}

		// cloud task にTopic告知タスクを登録
		err = AddTopicTaskToCloudTasks(db, task, videos)
		if err != nil {
			return err
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

	return nil
}

// 新しい動画の取得漏れがないかRSSを使って確認
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

	notExistRssVIDs, err := db.NotExistsVideoID(rssVideoIDs)
	if err != nil {
		return err
	}

	if len(notExistRssVIDs) == 0 {
		return nil
	}

	slog.Info("RssFeed",
		slog.String("notExistRssVIDs", strings.Join(notExistRssVIDs, ",")),
	)

	for _, vid := range notExistRssVIDs {
		// 5分後に
		err := task.CreateExistCheckTask(vid)
		if err != nil {
			slog.Error(err.Error(),
				slog.String("video_id", vid),
			)
			return err
		}
	}

	return nil
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

// Topic通知
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
		slog.Warn("deleted video",
			slog.String("video_id", vid),
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
		slog.Error(err.Error())
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

func CheckExistVideo(vid string) error {
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}
	defer db.Close()

	vids, err := db.NotExistsVideoID([]string{vid})
	if err != nil {
		return err
	}

	// DBに登録されている動画IDだった場合
	if len(vids) == 0 {
		return nil
	}

	// DBに登録されていない動画IDだった場合
	err = NewMail().Subject("検証 動画ないよ").Id(vid).Title(vid).Send()
	if err != nil {
		slog.Error(err.Error())
		return err
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
		err := task.CreateSongTask(v)
		if err != nil {
			return err
		}
	}
	return nil
}

// cloud task にTopic告知タスクを登録
// 実際のプッシュ通知はユーザに登録されているTopicのみ通知する
func AddTopicTaskToCloudTasks(db *DB, task *Task, videos []youtube.Video) error {
	// Topic全件取得
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

	return nil
}

func GetNewVideoIDs(yt *Youtube, db *DB, oldPlaylists map[string]Playlist, newPlaylists map[string]Playlist) ([]string, error) {
	// 新しくアップロードされた動画ID
	var newVIDs []string
	
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
			// 動画が削除されても、動画がアップロードされている可能性もあるためcontinueしない
			slog.Warn("動画が削除されている可能性があります")
		}

		// 更新されたプレイリストの動画IDリストを取得
		vids, err := yt.PlaylistItems(pid)
		if err != nil {
			return nil, err
		}

		// DBに登録されていない動画のみにフィルター
		notExistsVIDs, err := db.NotExistsVideoID(vids)
		if err != nil {
			return nil, err
		}

		// プレイリストに新しくアップロードされた動画があった場合
		if len(notExistsVIDs) != 0 {
			newVIDs = append(newVIDs, notExistsVIDs...)
			continue
		}

		// プレイリストから新しい動画が見つからなかった場合
		// RSSから過去30分間にアップロードされた動画IDを取得
		rssVideoIDs, err := yt.RssFeed([]string{pid})
		if err != nil {
			return nil, err
		}

		// RSSに新しくアップロードされた動画があった場合
		if len(rssVideoIDs) != 0 {
			newVIDs = append(newVIDs, rssVideoIDs...)
			continue
		}

		// 新しくアップロードされた動画が見つからなかった場合
		slog.Warn("動画が見つかりませんでした",
			slog.String("playlist_id", pid),
		)
	}

	return newVIDs, nil
}
