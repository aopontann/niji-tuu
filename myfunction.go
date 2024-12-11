package nsa

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("check", func(w http.ResponseWriter, r *http.Request) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger) // 以降、JSON形式で出力される。

		err := CheckNewVideoJob()
		if err != nil {
			handleError(w, err, "CheckNewVideoJob")
		}
	})

	functions.HTTP("check-rss", func(w http.ResponseWriter, r *http.Request) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger) // 以降、JSON形式で出力される。

		err := CheckNewVideoJobWithRSS()
		if err != nil {
			handleError(w, err, "CheckNewVideoJobWithRSS")
		}
	})

	functions.HTTP("exist-check", func(w http.ResponseWriter, r *http.Request) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger) // 以降、JSON形式で出力される。

		var b ExistCheckTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error("NewDecoder error",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, "リクエストボディが不正です", http.StatusBadRequest)
			return
		}

		if b.ID == "" {
			http.Error(w, "ID is required", http.StatusBadRequest)
			return
		}

		err := CheckExistVideo(b.ID)
		if err != nil {
			handleError(w, err, "CheckExistVideo")
		}
	})

	functions.HTTP("song", func(w http.ResponseWriter, r *http.Request) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger) // 以降、JSON形式で出力される。

		var b SongTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error("NewDecoder error",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, "リクエストボディが不正です", http.StatusBadRequest)
			return
		}

		err := SongVideoAnnounceJob(b.ID)
		if err != nil {
			handleError(w, err, "SongVideoAnnounceJob")
		}
	})

	functions.HTTP("topic", func(w http.ResponseWriter, r *http.Request) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger) // 以降、JSON形式で出力される。

		var b TopicTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error("NewDecoder error",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, "リクエストボディが不正です", http.StatusBadRequest)
			return
		}

		err := TopicAnnounceJob(b.VID, b.TID)
		if err != nil {
			handleError(w, err, "TopicAnnounceJob")
		}
	})
}

func handleError(w http.ResponseWriter, err error, operation string) {
	slog.Error(operation,
		slog.String("severity", "ERROR"),
		slog.String("message", err.Error()),
	)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
