package nsa

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("demo", func(w http.ResponseWriter, r *http.Request) {
		var b SongTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error("NewDecoder error",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
			return
		}
		slog.Info("cloud task demo!!!",
			slog.String("id", b.ID),
		)
	})

	functions.HTTP("check", func(w http.ResponseWriter, r *http.Request) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger) // 以降、JSON形式で出力される。

		err := CheckNewVideoJob()
		if err != nil {
			slog.Error("CheckNewVideoJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
			return
		}

		err := SongVideoAnnounceJob(b.ID)
		if err != nil {
			slog.Error("SongVideoAnnounceJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
			return
		}

		err := TopicAnnounceJob(b.VID, b.TID)
		if err != nil {
			slog.Error("TopicAnnounceJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
