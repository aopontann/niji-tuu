package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aopontann/niji-tuu/internal"
	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"
	"google.golang.org/api/youtube/v3"
)

const (
	songKeywordsRegexPattern   = ".*cover|歌って|歌わせて|original song|オリジナル曲|オリジナル楽曲|オリジナルソング|mv|music video.*"
	ignoreKeywordsRegexPattern = ".*切り抜き|ラジオ|くろなん.*"
)

const (
	roleID    = "1359103811339161701"
	ChannelID = "1350460034865430592"
)

var db *sqlx.DB

func main() {
	db = internal.NewDB(os.Getenv("DSN"))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /song/check", func(w http.ResponseWriter, r *http.Request) {
		vid := r.FormValue("v")
		if vid == "" {
			msg := "クエリパラメータ v が指定されていません"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		// checkのときのみ","区切りで動画IDが送られる
		vids := strings.Split(vid, ",")
		err := SongJob(vids)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// チェックなしで登録
	mux.HandleFunc("POST /song", func(w http.ResponseWriter, r *http.Request) {
		vid := r.FormValue("v")
		if vid == "" {
			msg := "クエリパラメータ v が指定されていません"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		vids := strings.Split(vid, ",")
		yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		videos, err := yt.Videos(vids)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		err = RegisterNotifySongTask(videos)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("POST /song/notify/{type}", func(w http.ResponseWriter, r *http.Request) {
		vid := r.FormValue("v")
		if vid == "" {
			msg := "クエリパラメータ v が指定されていません"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		switch r.PathValue("type") {
		case "fcm":
			err := NotifySongOnFCM(vid)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		case "discord":
			err := NotifySongOnDiscord(vid)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		case "twitter":
			err := NotifySongOnTwitter(vid)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		case "instagram":
			err := NotifySongOnInstagram(vid)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		default:
			m := "存在しないパスを指定しています"
			slog.Warn(m)
			http.Error(w, m, http.StatusNotFound)
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

// SongJob 歌動画関連の処理のエントリーポイント
// ハンドラ関数から呼び出される
func SongJob(vids []string) error {
	// YouTube Data APIから動画情報を取得
	y, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	videos, err := y.Videos(vids)
	if err != nil {
		return err
	}

	// 指定された動画が歌動画かチェック
	songVideos, maybeSongVideos := CheckSong(videos)

	// 指定時刻に歌動画を告知するようにタスクを登録
	err = RegisterNotifySongTask(songVideos)
	if err != nil {
		return err
	}

	// 歌動画か判別できなかったものは動画情報をDiscordサーバに送信
	err = SendMaybeSongVideosForDiscord(maybeSongVideos)
	if err != nil {
		return err
	}

	return nil
}

// CheckSong 歌動画か解析
// 返り値の1つ目は歌動画であることが確定している動画
// 返り値の2つ目は歌動画であるか判断できない動画
func CheckSong(videos []youtube.Video) ([]youtube.Video, []youtube.Video) {
	songRegex, _ := regexp.Compile(songKeywordsRegexPattern)
	ignoreRegex, _ := regexp.Compile(ignoreKeywordsRegexPattern)

	var res []youtube.Video
	var maybeRes []youtube.Video
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
		// 小文字大文字を区別しないで比較するための処理
		t := strings.ToLower(v.Snippet.Title)
		// 除外したいキーワードがタイトルに含まれている場合
		if ignoreRegex.MatchString(t) {
			continue
		}
		// 歌動画によく含まれているキーワードがタイトルに含まれている場合
		// 含まれていない場合でも歌動画の可能性があるため、別のスライスに動画情報を格納
		if songRegex.MatchString(t) {
			res = append(res, v)
		} else {
			maybeRes = append(maybeRes, v)
		}
	}

	return res, maybeRes
}

func RegisterNotifySongTask(videos []youtube.Video) error {
	t, err := internal.NewTask()
	if err != nil {
		return err
	}

	for _, v := range videos {
		taskInfoFCM := &internal.TaskInfo{
			Video:      v,
			QueueID:    "song-queue",
			URL:        os.Getenv("HOST") + "/song/notify/fcm",
			MinutesAgo: time.Minute * 5,
		}
		taskInfoDiscord := &internal.TaskInfo{
			Video:      v,
			QueueID:    "song-queue",
			URL:        os.Getenv("HOST") + "/song/notify/discord",
			MinutesAgo: time.Hour * 1,
		}
		taskInfoTwitter := &internal.TaskInfo{
			Video:      v,
			QueueID:    "song-queue",
			URL:        os.Getenv("HOST") + "/song/notify/twitter",
			MinutesAgo: time.Minute * 5,
		}
		taskInfoInstagram := &internal.TaskInfo{
			Video:      v,
			QueueID:    "song-queue",
			URL:        os.Getenv("HOST") + "/song/notify/instagram",
			MinutesAgo: time.Minute * 5,
		}

		if err := t.Create(taskInfoFCM); err != nil {
			slog.Error(err.Error())
			return err
		}
		if err := t.Create(taskInfoDiscord); err != nil {
			slog.Error(err.Error())
			return err
		}
		if err := t.Create(taskInfoTwitter); err != nil {
			slog.Error(err.Error())
			return err
		}
		if err := t.Create(taskInfoInstagram); err != nil {
			slog.Error(err.Error())
			return err
		}
	}

	return nil
}

func SendMaybeSongVideosForDiscord(videos []youtube.Video) error {
	for _, video := range videos {
		body := []byte(fmt.Sprintf(`{"content": "https://www.youtube.com/watch?v=%s"}`, video.Id))
		resp, err := http.Post(
			os.Getenv("DISCORD_WEBHOOK_MAYBE_SONG"),
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			return err
		}
		if resp.StatusCode != 204 {
			slog.Error("204以外のステータスコードを取得しました",
				slog.Int("code", resp.StatusCode),
			)
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func NotifySongOnFCM(vid string) error {
	y, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	f := internal.NewFCM()

	// 動画か消されていないかチェック
	videos, err := y.Videos([]string{vid})
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
	var tokens []string
	err = db.Select(&tokens, "SELECT token FROM users WHERE song = 1")
	if err != nil {
		return err
	}

	title := videos[0].Snippet.Title
	thumbnail := videos[0].Snippet.Thumbnails.High.Url

	slog.Info("song-video-announce",
		slog.String("video_id", vid),
		slog.String("title", title),
	)

	err = f.Notification(
		"5分後に公開",
		tokens,
		&internal.NotificationVideo{
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

func NotifySongOnDiscord(vid string) error {
	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
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

func NotifySongOnTwitter(vid string) error {
	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
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

	t := internal.NewTwitter(
		os.Getenv("TWITTER_API_KEY"),
		os.Getenv("TWITTER_API_SECRET_KEY"),
		os.Getenv("TWITTER_ACCESS_TOKEN"),
		os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
	)

	err = t.Notification(&internal.NotificationVideo{
		ID:        video.Id,
		Title:     video.Snippet.Title,
		Thumbnail: video.Snippet.Thumbnails.Maxres.Url,
	})
	if err != nil {
		return err
	}

	return nil
}

func NotifySongOnInstagram(vid string) error {
	id := os.Getenv("INSTAGRAM_ID")
	token := os.Getenv("INSTAGRAM_ACCESS_TOKEN")
	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
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

	insta := internal.NewInstagram(id, token)
	err = insta.Notification(&internal.NotificationVideo{
		ID:        vid,
		Title:     video.Snippet.Title,
		Thumbnail: video.Snippet.Thumbnails.Maxres.Url,
	})
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return nil
}
