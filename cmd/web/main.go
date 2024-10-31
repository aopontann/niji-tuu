package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/cors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	nsa "github.com/aopontann/nijisanji-songs-announcement"
)

type ReqBody struct {
	Status bool `json:"status"`
}
type ReqBodyTopic struct {
	TopicID string `json:"topic_id"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()
	config, err := pgx.ParseConfig(os.Getenv("DSN"))
	if err != nil {
		panic(err)
	}
	sqldb := stdlib.OpenDB(*config)
	db := bun.NewDB(sqldb, pgdialect.New())

	mux := http.NewServeMux()
	mux.HandleFunc("/api/song", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-Type", "application/json")
		if len(r.Header["Authorization"]) == 0 {
			http.Error(w, "NG", http.StatusBadRequest)
			return
		}
		token := strings.Split(r.Header["Authorization"][0], " ")[1]

		var check bool
		if r.Method == http.MethodGet {
			err := db.NewSelect().Column("song").Table("users").Where("token = ?", token).Scan(ctx, &check)
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
				slog.Error("NewDecoder error",
					slog.String("severity", "ERROR"),
					slog.String("message", err.Error()),
				)
				http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
				return
			}

			slog.Info("POST",
				slog.String("severity", "INFO"),
				slog.String("token", token),
				slog.String("User-Agent", r.Header["User-Agent"][0]),
			)
			_, err := db.NewInsert().
				Model(&nsa.User{Token: token, Song: b.Status}).
				On("CONFLICT (token) DO UPDATE").
				Set("song = EXCLUDED.song").
				Exec(ctx)
			if err != nil {
				slog.Error("Upsert error",
					slog.String("severity", "ERROR"),
					slog.String("message", err.Error()),
				)
				http.Error(w, "処理に失敗しました", http.StatusInternalServerError)
				return
			}
			w.Write([]byte("OK!!"))
		}
	})

	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-Type", "application/json")
		if len(r.Header["Authorization"]) == 0 {
			http.Error(w, "NG", http.StatusBadRequest)
			return
		}
		token := strings.Split(r.Header["Authorization"][0], " ")[1]

		var check bool
		if r.Method == http.MethodGet {
			err := db.NewSelect().Column("info").Table("users").Where("token = ?", token).Scan(ctx, &check)
			if err == sql.ErrNoRows {
				http.Error(w, fmt.Sprintf(`{"status":%s}`, err), http.StatusNotFound)
				return
			}
			if err != nil {
				slog.Info("POST",
					slog.String("severity", "INFO"),
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
				slog.Error("NewDecoder error",
					slog.String("severity", "ERROR"),
					slog.String("message", err.Error()),
				)
				http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
				return
			}

			slog.Info("POST",
				slog.String("severity", "INFO"),
				slog.String("token", token),
				slog.String("User-Agent", r.Header["User-Agent"][0]),
			)
			_, err := db.NewInsert().
				Model(&nsa.User{Token: token, Info: b.Status}).
				On("CONFLICT (token) DO UPDATE").
				Set("info = EXCLUDED.info").
				Exec(ctx)
			if err != nil {
				slog.Error("Upsert error",
					slog.String("severity", "ERROR"),
					slog.String("message", err.Error()),
				)
				http.Error(w, "処理に失敗しました", http.StatusInternalServerError)
				return
			}
			w.Write([]byte("OK!!"))
		}
	})

	mux.HandleFunc("/api/topic", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-Type", "application/json")
		if len(r.Header["Authorization"]) == 0 {
			http.Error(w, "NG", http.StatusBadRequest)
			return
		}
		token := strings.Split(r.Header["Authorization"][0], " ")[1]

		if r.Method == http.MethodGet {
			var topics []nsa.Topic
			err := db.NewSelect().
				ColumnExpr("topic_id AS id, topics.name AS name").
				Table("user_topics").
				Join("JOIN topics ON topic_id = topics.id").
				Where("user_token = ?", token).
				Scan(ctx, &topics)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"status":%s}`, err), http.StatusInternalServerError)
				return
			}
			if len(topics) == 0 {
				http.Error(w, fmt.Sprintf(`{"status":%s}`, err), http.StatusNotFound)
				return
			}
			s, _ := json.Marshal(topics)
			w.Write(s)
			return
		}

		fcm := nsa.NewFCM()
		var b ReqBodyTopic
		if err = json.NewDecoder(r.Body).Decode(&b); err != nil {
			slog.Error("NewDecoder error",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			http.Error(w, "リクエストボディが不正です", http.StatusInternalServerError)
			return
		}

		registerCount, err := db.NewSelect().Table("user_topics").Where("user_token = ?", token).Count(ctx)
		if err != nil {
			Error(w, err)
			return
		}
		if registerCount >= 5 {
			http.Error(w, "キーワードは5個までしか追加できません", http.StatusBadRequest)
			return
		}

		var topicName string
		_, err = db.NewSelect().Column("name").Table("topics").Where("id = ?", b.TopicID).Exec(ctx, &topicName)
		if err != nil {
			Error(w, err)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			Error(w, err)
			return
		}

		if r.Method == http.MethodPost {
			slog.Info("POST",
				slog.String("severity", "INFO"),
				slog.String("token", token),
				slog.String("User-Agent", r.Header["User-Agent"][0]),
			)

			data := nsa.UserTopic{
				UserToken: token,
				TopicID:   b.TopicID,
			}

			_, err := tx.NewInsert().Model(&data).Ignore().Exec(ctx)
			if err != nil {
				Error(w, err)
				return
			}

			err = fcm.SetTopic(token, topicName)
			if err != nil {
				tx.Rollback()
				Error(w, err)
				return
			}
		}

		if r.Method == http.MethodDelete {
			slog.Info("DELETE",
				slog.String("severity", "INFO"),
				slog.String("token", token),
				slog.String("User-Agent", r.Header["User-Agent"][0]),
			)
			_, err := tx.NewDelete().Model((*nsa.UserTopic)(nil)).Where("user_token = ? AND topic_id = ?", token, b.TopicID).Exec(ctx)
			if err != nil {
				Error(w, err)
				return
			}

			err = fcm.DeleteTopic(token, topicName)
			if err != nil {
				tx.Rollback()
				Error(w, err)
				return
			}
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
	})

	mux.HandleFunc("/api/topic/list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-Type", "application/json")

		var topics []nsa.Topic
		if r.Method == http.MethodGet {
			err := db.NewSelect().Column("id", "name").Model(&topics).Scan(ctx)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"status":%s}`, err), http.StatusInternalServerError)
				return
			}
			s, _ := json.Marshal(topics)
			w.Write(s)
			// w.Write([]byte(fmt.Sprintf(`{"status":["%s"]}`, strings.Join(topics, `","`))))
		}

	})

	mux.HandleFunc("/api/unsubscription", func(w http.ResponseWriter, r *http.Request) {
		if len(r.Header["Authorization"]) == 0 {
			http.Error(w, "NG", http.StatusBadRequest)
			return
		}
		token := strings.Split(r.Header["Authorization"][0], " ")[1]
		if r.Method == http.MethodPost {
			tx, err := db.Begin()
			if err != nil {
				Error(w, err)
				return
			}

			_, err = tx.NewDelete().Model((*nsa.UserTopic)(nil)).Where("user_token = ?", token).Exec(ctx)
			if err != nil {
				Error(w, err)
				return
			}

			_, err = tx.NewDelete().Model((*nsa.User)(nil)).Where("token = ?", token).Exec(ctx)
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
	// CORS レスポンスヘッダーの追加
	c := cors.AllowAll()
	handler := c.Handler(mux)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func Error(w http.ResponseWriter, err error) {
	slog.Error("insert error",
		slog.String("severity", "ERROR"),
		slog.String("message", err.Error()),
	)
	http.Error(w, "処理に失敗しました", http.StatusInternalServerError)
}
