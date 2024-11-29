package nsa

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestNotExistsVideos(t *testing.T) {
	bunDB := setup()
	defer bunDB.Close()
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err)
	}
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}
	ctx := context.Background()

	videos := []Video{
		{ID: "SDr4sxCuMf0", Title: "MV", Duration: "PT4M", Song: false, Viewers: 0, Content: "upcoming", StartTime: time.Date(2024, 6, 27, 12, 10, 0, 0, time.UTC)},
	}
	_, err = bunDB.NewInsert().Model(&videos).Exec(ctx)
	if err != nil {
		t.Error(err)
	}

	ytVideos, err := yt.Videos([]string{"SDr4sxCuMf0", "YIQFuRXF3tQ"})
	if err != nil {
		t.Error(err)
	}

	notExistsVideos, err := db.NotExistsVideos(ytVideos)
	if err != nil {
		t.Error(err)
	}

	for _, v := range notExistsVideos {
		t.Log(v.Id)
	}

	if notExistsVideos[0].Id != "YIQFuRXF3tQ" {
		t.Error(err)
	}

	_, err = bunDB.NewDelete().Model(&videos).WherePK().Exec(ctx)
	if err != nil {
		t.Error(err)
	}
}

func setup() *bun.DB {
	config, err := pgx.ParseConfig(os.Getenv("DSN"))
	if err != nil {
		panic(err)
	}
	sqldb := stdlib.OpenDB(*config)
	return bun.NewDB(sqldb, pgdialect.New())
}

func TestGetTopicWhereUserRegister(t *testing.T) {
	godotenv.Load(".env")
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err)
	}

	topicID := 4
	topic, err := db.getTopicWhereUserRegister(topicID)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Log("empty")
		}
		t.Error(err)
	}
	t.Log(topic)
}

func TestUpdatePlaylistItem(t *testing.T) {
	godotenv.Load(".env")
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err.Error())
	}
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	pids, err := db.PlaylistIDs()
	if err != nil {
		t.Fatal(err.Error())
	}

	newPlaylists, err := yt.Playlists(pids)
	if err != nil {
		t.Fatal(err.Error())
	}

	// トランザクション開始
	ctx := context.Background()
	tx, err := db.Service.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	// DBのプレイリスト動画数を更新
	err = db.UpdatePlaylistItem(tx, newPlaylists)
	if err != nil {
		tx.Rollback()
		t.Fatal(err.Error())
	}

	err = tx.Commit()
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestGetAllTopics(t *testing.T) {
	godotenv.Load(".env")
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err.Error())
	}

	topics, err := db.getAllTopics()
	if err != nil {
		t.Fatal(err.Error())
	}

	for _, topic := range topics {
		t.Log(topic)
	}

}
