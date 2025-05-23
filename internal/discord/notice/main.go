package discordnotice

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/aopontann/niji-tuu/internal/common/db"
	"github.com/aopontann/niji-tuu/internal/common/youtube"
	"github.com/bwmarrin/discordgo"
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

	err := DiscordAnnounceJob(vid)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func DiscordAnnounceJob(vid string) error {
	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	cdb, err := db.NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}
	defer cdb.Close()
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

	keywords, err := cdb.GetKeywords()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		slog.Error(err.Error())
		return err
	}

	for _, keyword := range keywords {
		// 小文字に統一してから一致チェック
		titleLower := strings.ToLower(title)

		// キーワードに一致するか
		words := strings.Join(keyword.Include, "|")
		wordsLower := strings.ToLower(words)
		regPattern := ".*" + wordsLower + ".*"
		regex, _ := regexp.Compile(regPattern)
		if !regex.MatchString(titleLower) {
			continue
		}

		// 除外するキーワードに一致した場合通知しない
		words = strings.Join(keyword.Ignore, "|")
		wordsLower = strings.ToLower(words)
		regPattern = ".*" + wordsLower + ".*"
		regex, _ = regexp.Compile(regPattern)
		if len(keyword.Ignore) != 0 && regex.MatchString(titleLower) {
			continue
		}

		// キーワードに一致した場合
		content := fmt.Sprintf("<@&%s>\nhttps://www.youtube.com/watch?v=%s", keyword.RoleID, vid)
		_, err := discord.ChannelMessageSend(keyword.ChannelID, content)
		if err != nil {
			return err
		}
	}

	return nil
}
