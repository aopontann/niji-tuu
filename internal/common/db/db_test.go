package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aopontann/niji-tuu/internal/common/youtube"
)

func TestUpdateVtubers(t *testing.T) {
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
	ctx := context.Background()
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

	err = db.SaveVideos(videos, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	count, err := db.Service.NewSelect().Model((*Video)(nil)).Count(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
	if count != 2 {
		t.Fatalf("expected 2 videos, got %d", count)
	}
}

func TestNotExistsVideoID(t *testing.T) {
	db, err := NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err)
	}

	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	videos, err := yt.Videos([]string{"EgaXyUcsM48"})
	if err != nil {
		t.Fatal(err.Error())
	}

	err = db.SaveVideos(videos, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	vids, err := db.NotExistsVideoID([]string{"EgaXyUcsM48", "QM68LWmAtJQ"})
	if err != nil {
		t.Fatal(err)
	}

	if len(vids) == 0 || vids[0] != "QM68LWmAtJQ" {
		t.Fatalf("expected QM68LWmAtJQ, got %s", vids[0])
	}
}

func TestGetRoles(t *testing.T) {
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
