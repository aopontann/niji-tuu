package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/aopontann/niji-tuu/internal/common/db"
)

type ReqBody struct {
	Status bool `json:"status"`
}
type ReqBodyTopic struct {
	TopicID string `json:"topic_id"`
}

//go:embed dist/*
var dist embed.FS

func main() {
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

	ctx := context.Background()
	config, err := pgx.ParseConfig(os.Getenv("DSN"))
	if err != nil {
		panic(err)
	}
	sqldb := stdlib.OpenDB(*config)
	cdb := bun.NewDB(sqldb, pgdialect.New())

	dist, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	http.Handle("/", http.FileServer(http.FS(dist)))

	http.HandleFunc("/api/song", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-Type", "application/json")
		if len(r.Header["Authorization"]) == 0 {
			http.Error(w, "NG", http.StatusBadRequest)
			return
		}
		token := strings.Split(r.Header["Authorization"][0], " ")[1]

		var check bool
		if r.Method == http.MethodGet {
			err := cdb.NewSelect().Column("song").Table("users").Where("token = ?", token).Scan(ctx, &check)
			if err == sql.ErrNoRows {
				http.Error(w, fmt.Sprintf(`{"status":%s}`, err), http.StatusNotFound)
				return
			}
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"status":%s}`, err), http.StatusInternalServerError)
				return
			}
			w.Write([]byte(fmt.Sprintf(`{"status":%t}`, check)))
		}

		if r.Method == http.MethodPost {
			var b ReqBody
			if err = json.NewDecoder(r.Body).Decode(&b); err != nil {
				slog.Error(err.Error())
				http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
				return
			}

			slog.Info("POST",
				slog.String("token", token),
				slog.String("User-Agent", r.Header["User-Agent"][0]),
			)
			_, err := cdb.NewInsert().
				Model(&db.User{Token: token, Song: b.Status}).
				On("CONFLICT (token) DO UPDATE").
				Set("song = EXCLUDED.song").
				Exec(ctx)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, "処理に失敗しました", http.StatusInternalServerError)
				return
			}
			w.Write([]byte("OK!!"))
		}
	})

	http.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-Type", "application/json")
		if len(r.Header["Authorization"]) == 0 {
			http.Error(w, "NG", http.StatusBadRequest)
			return
		}
		token := strings.Split(r.Header["Authorization"][0], " ")[1]

		var check bool
		if r.Method == http.MethodGet {
			err := cdb.NewSelect().Column("info").Table("users").Where("token = ?", token).Scan(ctx, &check)
			if err == sql.ErrNoRows {
				http.Error(w, fmt.Sprintf(`{"status":%s}`, err), http.StatusNotFound)
				return
			}
			if err != nil {
				slog.Info("POST",
					slog.String("token", token),
					slog.String("User-Agent", r.Header["User-Agent"][0]),
				)
				return
			}
			w.Write([]byte(fmt.Sprintf(`{"status":%t}`, check)))
		}

		if r.Method == http.MethodPost {
			var b ReqBody
			if err = json.NewDecoder(r.Body).Decode(&b); err != nil {
				slog.Error(err.Error())
				http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
				return
			}

			slog.Info("POST",
				slog.String("token", token),
				slog.String("User-Agent", r.Header["User-Agent"][0]),
			)
			_, err := cdb.NewInsert().
				Model(&db.User{Token: token, Info: b.Status}).
				On("CONFLICT (token) DO UPDATE").
				Set("info = EXCLUDED.info").
				Exec(ctx)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, "処理に失敗しました", http.StatusInternalServerError)
				return
			}
			w.Write([]byte("OK!!"))
		}
	})

	http.HandleFunc("/api/unsubscription", func(w http.ResponseWriter, r *http.Request) {
		if len(r.Header["Authorization"]) == 0 {
			http.Error(w, "NG", http.StatusBadRequest)
			return
		}
		token := strings.Split(r.Header["Authorization"][0], " ")[1]
		if r.Method == http.MethodPost {
			tx, err := cdb.Begin()
			if err != nil {
				Error(w, err)
				return
			}

			_, err = tx.NewDelete().Model((*db.User)(nil)).Where("token = ?", token).Exec(ctx)
			if err != nil {
				Error(w, err)
				return
			}

			err = retry.Do(
				tx.Commit,
				retry.Attempts(3),
				retry.Delay(2*time.Second),
			)
			if err != nil {
				Error(w, err)
				return
			}
			w.Write([]byte("OK!!"))
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func Error(w http.ResponseWriter, err error) {
	slog.Error(err.Error())
	http.Error(w, "処理に失敗しました", http.StatusInternalServerError)
}
