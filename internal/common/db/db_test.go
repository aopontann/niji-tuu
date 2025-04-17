package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/aopontann/niji-tuu/internal/youtube"
)

func TestUpdateVtubers(t *testing.T) {
	godotenv.Load(".env")
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err.Error())
	}

	var vtubers []Vtuber
	ctx := context.Background()
	err = db.Service.NewSelect().Model(&vtubers).Where("id = ?", "UC0g1AE0DOjBYnLhkgoRWN1w").Scan(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}

	vtubers[0].ItemCount = 9999

	// DBのプレイリスト動画数を更新
	err = db.UpdateVtubers(vtubers, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestSaveVideo(t *testing.T) {
	godotenv.Load(".env.dev")
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err.Error())
	}
	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	videos, err := yt.Videos([]string{"EgaXyUcsM48", "QM68LWmAtJQ"})
	if err != nil {
		t.Fatal(err.Error())
	}

	// トランザクション開始
	ctx := context.Background()
	tx, err := db.Service.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.SaveVideos(videos, &tx)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = tx.Commit()
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNotExistsVideoID(t *testing.T) {
	godotenv.Load(".env")
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err)
	}
	vids, err := db.NotExistsVideoID([]string{"aaa", "bbb"})
	if err != nil {
		t.Fatal(err)
	}

	log.Println(vids)
}

func TestGetRoles(t *testing.T) {
	godotenv.Load(".env")
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err)
	}
	roles, err := db.GetRoles()
	if err != nil {
		t.Fatal(err)
	}

	for _, role := range roles {
		fmt.Println(role)
	}
}
