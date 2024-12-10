package nsa

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

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

	// DBのプレイリスト動画数を更新
	err = db.UpdatePlaylistItem(newPlaylists)
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
