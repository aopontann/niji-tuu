package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	nsa "github.com/aopontann/nijisanji-songs-announcement"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger) // 以降、JSON形式で出力される。

	if os.Getenv("ENV") != "prod" {
		godotenv.Load(".env")
	}

	config, err := pgx.ParseConfig(os.Getenv("DSN"))
	if err != nil {
		panic(err)
	}
	sqldb := stdlib.OpenDB(*config)
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	job := nsa.NewJobs(
		os.Getenv("YOUTUBE_API_KEY"),
		db,
	)

	http.HandleFunc("/v2/check", func(w http.ResponseWriter, r *http.Request) {
		err := job.CheckNewVideoJob()
		if err != nil {
			slog.Error("CheckNewVideoJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/v2/song", func(w http.ResponseWriter, r *http.Request) {
		err := job.SongVideoAnnounceJob()
		if err != nil {
			slog.Error("SongVideoAnnounceJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/v2/keyword", func(w http.ResponseWriter, r *http.Request) {
		err := job.KeywordAnnounceJob()
		if err != nil {
			slog.Error("KeywordAnnounceJob",
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
