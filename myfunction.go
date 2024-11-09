package nsa

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func init() {
	functions.HTTP("check", func(w http.ResponseWriter, r *http.Request) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger) // 以降、JSON形式で出力される。

		config, err := pgx.ParseConfig(os.Getenv("DSN"))
		if err != nil {
			panic(err)
		}
		sqldb := stdlib.OpenDB(*config)
		db := bun.NewDB(sqldb, pgdialect.New())
		defer db.Close()

		job := NewJobs(
			os.Getenv("YOUTUBE_API_KEY"),
			db,
		)

		err = job.CheckNewVideoJob()
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

		config, err := pgx.ParseConfig(os.Getenv("DSN"))
		if err != nil {
			panic(err)
		}
		sqldb := stdlib.OpenDB(*config)
		db := bun.NewDB(sqldb, pgdialect.New())
		defer db.Close()

		job := NewJobs(
			os.Getenv("YOUTUBE_API_KEY"),
			db,
		)

		err = job.SongVideoAnnounceJob()
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

		config, err := pgx.ParseConfig(os.Getenv("DSN"))
		if err != nil {
			panic(err)
		}
		sqldb := stdlib.OpenDB(*config)
		db := bun.NewDB(sqldb, pgdialect.New())
		defer db.Close()

		job := NewJobs(
			os.Getenv("YOUTUBE_API_KEY"),
			db,
		)

		err = job.KeywordAnnounceJob()
		if err != nil {
			slog.Error("KeywordAnnounceJob",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
