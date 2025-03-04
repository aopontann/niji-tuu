package nsa

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

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

	// DBのプレイリスト動画数を更新
	err = db.UpdatePlaylistItem(newPlaylists)
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
