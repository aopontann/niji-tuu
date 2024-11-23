package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"

	nsa "github.com/aopontann/nijisanji-songs-announcement"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger) // 以降、JSON形式で出力される。

	if os.Getenv("ENV") != "prod" {
		godotenv.Load(".env")
	}

	http.HandleFunc("/v2/check", func(w http.ResponseWriter, r *http.Request) {
		err := nsa.CheckNewVideoJob()
		if err != nil {
			slog.Error("CheckNewVideoJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/v2/song", func(w http.ResponseWriter, r *http.Request) {
		var b nsa.SongTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error("NewDecoder error",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
			return
		}
		err := nsa.SongVideoAnnounceJob(b.ID)
		if err != nil {
			slog.Error("SongVideoAnnounceJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/v2/keyword", func(w http.ResponseWriter, r *http.Request) {
		var b nsa.TopicTaskReqBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error("NewDecoder error",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
			return
		}
		err := nsa.TopicAnnounceJob(b.VID, b.TID)
		if err != nil {
			slog.Error("TopicAnnounceJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Start HTTP server.
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
