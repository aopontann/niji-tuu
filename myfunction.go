package nsa

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

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

	functions.HTTP("check-rss", func(w http.ResponseWriter, r *http.Request) {
		err := CheckNewVideoJobWithRSS()
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	functions.HTTP("exist-check", func(w http.ResponseWriter, r *http.Request) {
		var b ExistCheckTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error(err.Error())
			http.Error(w, "リクエストボディが不正です", http.StatusBadRequest)
			return
		}

		if b.ID == "" {
			http.Error(w, "ID is required", http.StatusBadRequest)
			return
		}

		err := CheckExistVideo(b.ID)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	functions.HTTP("song", func(w http.ResponseWriter, r *http.Request) {
		var b SongTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error(err.Error())
			http.Error(w, "リクエストボディが不正です", http.StatusBadRequest)
			return
		}

		err := SongVideoAnnounceJob(b.ID)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	functions.HTTP("topic", func(w http.ResponseWriter, r *http.Request) {
		var b TopicTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error(err.Error())
			http.Error(w, "リクエストボディが不正です", http.StatusBadRequest)
			return
		}

		err := TopicAnnounceJob(b.VID, b.TID)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
