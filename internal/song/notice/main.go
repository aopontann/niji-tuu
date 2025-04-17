package songnotice

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/aopontann/niji-tuu/internal/common/db"
	"github.com/aopontann/niji-tuu/internal/common/fcm"
	"github.com/aopontann/niji-tuu/internal/common/youtube"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		msg := "POSTメソッドでリクエストしてください"
		slog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	vid := r.FormValue("v")
	if vid == "" {
		msg := "クエリパラメータ v が指定されていません"
		slog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	err := SongVideoAnnounceJob(vid)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 歌動画通知
func SongVideoAnnounceJob(vid string) error {
	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	cdb, err := db.NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}
	defer cdb.Close()
	cfcm := fcm.NewFCM()

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
	tokens, err := cdb.GetSongTokens()
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
	body := []byte(fmt.Sprintf(`{"content": "https://www.youtube.com/watch?v=%s"}`, vid))
	resp, err := http.Post(
		os.Getenv("DISCORD_WEBHOOK_SONG"),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	resp.Body.Close()

	err = cfcm.Notification(
		"5分後に公開",
		tokens,
		&fcm.NotificationVideo{
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
