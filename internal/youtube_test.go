package internal

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if os.Getenv("ENV") != "prod" {
		log.Println("Loading environmental variables...")
		if err := godotenv.Load("../.env.dev"); err != nil {
			log.Fatalln("failed to load env variables", err)
		}
	}
	m.Run()
}

func TestVideos(t *testing.T) {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Error(err)
	}

	vids := []string{"6uamOZjDubU"}
	res, err := yt.Videos(vids)
	if err != nil {
		t.Error(err)
	}
	for _, v := range res {
		log.Println(v)
	}
}

func TestSearch(t *testing.T) {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Error(err)
	}

	cids := []string{"UC-6rZgmxZSIbq786j3RD5ow"}
	res, err := yt.Search(cids)
	if err != nil {
		t.Error(err)
	}
	for _, v := range res {
		log.Println(v)
	}
}
