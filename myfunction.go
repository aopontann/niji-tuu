package nsa

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	// Cloud Logging用のログ設定
	ops := slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				a.Key = "severity"
				level := a.Value.Any().(slog.Level)
				if level == slog.LevelWarn {
					a.Value = slog.StringValue("WARNING")
				}
			}

			return a
		},
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &ops))
	slog.SetDefault(logger)

	functions.HTTP("check", func(w http.ResponseWriter, r *http.Request) {
		err := CheckNewVideoJob()
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	functions.HTTP("song-task", func(w http.ResponseWriter, r *http.Request) {
		vids := r.FormValue("v")
		if vids == "" {
			msg := "クエリパラメータ v が指定されていません"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		err := SongVideoCheck(strings.Split(vids, ","))
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	functions.HTTP("discord-task", func(w http.ResponseWriter, r *http.Request) {
		vids := r.FormValue("v")
		if vids == "" {
			msg := "クエリパラメータ v が指定されていません"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		err := CreateTaskToNoficationByDiscord(strings.Split(vids, ","))
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	functions.HTTP("notice-song", func(w http.ResponseWriter, r *http.Request) {
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
	})

	functions.HTTP("notice-discord", func(w http.ResponseWriter, r *http.Request) {
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
	})

	functions.HTTP("discord-bot", DiscordWebhook)
}
