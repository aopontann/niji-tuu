package songnotice

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/aopontann/niji-tuu/internal/common/db"
	"github.com/aopontann/niji-tuu/internal/common/fcm"
	"github.com/aopontann/niji-tuu/internal/common/youtube"
	"github.com/bwmarrin/discordgo"
)

const (  
    roleID    = "1359103811339161701"  
    ChannelID = "1350460034865430592"  
) 

func HandlerFCM(w http.ResponseWriter, r *http.Request) {
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

func HandlerDiscord(w http.ResponseWriter, r *http.Request) {
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

	err := NotifyFromDiscord(vid)
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

// discordから歌動画を通知
func NotifyFromDiscord(vid string) error {
	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}

	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		return err
	}

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

	video := videos[0]

	slog.Info("song-video-announce",
		slog.String("video_id", video.Id),
		slog.String("title", video.Snippet.Title),
	)

	// discordから通知
	content := fmt.Sprintf("<@&%s>\nhttps://www.youtube.com/watch?v=%s", roleID, vid)
	_, err = discord.ChannelMessageSend(ChannelID, content)
	if err != nil {
		return err
	}

	return nil
}
