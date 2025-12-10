package main

import (
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

var db *sqlx.DB

func main() {
	db = internal.NewDB(os.Getenv("DSN"))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /keyword/check", func(w http.ResponseWriter, r *http.Request) {
		vid := r.FormValue("v")
		if vid == "" {
			msg := "クエリパラメータ v が指定されていません"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		// checkのときのみ","区切りで動画IDが送られる
		vids := strings.Split(vid, ",")
		err := KeywordJob(vids)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("POST /keyword/notify/discord", func(w http.ResponseWriter, r *http.Request) {
		vid := r.FormValue("v")
		if vid == "" {
			msg := "クエリパラメータ v が指定されていません"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		err := NotifyKeywordOnDiscord(vid)
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

// KeywordJob キーワード関連の処理のエントリーポイント
// ハンドラ関数から呼び出される
func KeywordJob(vids []string) error {
	// YouTube Data APIから動画情報を取得
	y, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	videos, err := y.Videos(vids)
	if err != nil {
		return err
	}

	// 今後何か事前にチェックしたいときに使用する
	// checkedVideos := Check(videos)

	// 指定時刻に動画を告知するようにタスクを登録
	err = RegisterNotifyTask(videos)
	if err != nil {
		return err
	}

	return nil
}

func RegisterNotifyTask(videos []youtube.Video) error {
	t, err := internal.NewTask()
	if err != nil {
		return err
	}

	for _, v := range videos {
		taskInfoDiscord := &internal.TaskInfo{
			Video:      v,
			QueueID:    "keyword-queue",
			URL:        os.Getenv("HOST") + "/keyword/notify/discord",
			MinutesAgo: time.Hour * 1,
		}
		if err := t.Create(taskInfoDiscord); err != nil {
			slog.Error(err.Error())
			return err
		}
	}

	return nil
}

func NotifyKeywordOnDiscord(vid string) error {
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

	title := videos[0].Snippet.Title

	slog.Info("discord-announce",
		slog.String("video_id", vid),
		slog.String("title", title),
	)

	var keywords []internal.Keyword
	err = db.Select(&keywords, "SELECT * FROM keywords")
	if err != nil {
		return err
	}

	for _, keyword := range keywords {
		// 小文字に統一してから一致チェック
		titleLower := strings.ToLower(title)

		// キーワードに一致するか
		words := strings.ReplaceAll(keyword.InclusionList, ",", "|")
		wordsLower := strings.ToLower(words)
		regPattern := ".*" + wordsLower + ".*"
		regex, _ := regexp.Compile(regPattern)
		if !regex.MatchString(titleLower) {
			continue
		}

		// 除外するキーワードに一致した場合通知しない
		words = strings.ReplaceAll(keyword.ExclusionList, ",", "|")
		wordsLower = strings.ToLower(words)
		regPattern = ".*" + wordsLower + ".*"
		regex, _ = regexp.Compile(regPattern)
		if keyword.ExclusionList != "" && regex.MatchString(titleLower) {
			continue
		}

		// キーワードに一致した場合
		content := fmt.Sprintf("<@&%s>\n%s\nhttps://www.youtube.com/watch?v=%s", keyword.RoleID, title, vid)
		_, err := discord.ChannelMessageSend(keyword.ChannelID, content)
		if err != nil {
			return err
		}
	}

	return nil
}
