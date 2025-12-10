package main

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/aopontann/niji-tuu/internal"
	"github.com/joho/godotenv"
	"google.golang.org/api/youtube/v3"
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

func TestCheckSong(t *testing.T) {
	tests := []struct {
		Name                string
		filename            string
		SongVideoCount      int
		maybeSongVideoCount int
	}{
		{Name: "歌動画　公開前", filename: "testdata/youtube/video-before.json", SongVideoCount: 1, maybeSongVideoCount: 0},
		{Name: "歌動画　公開中", filename: "testdata/youtube/video-now.json", SongVideoCount: 1, maybeSongVideoCount: 0},
		{Name: "歌動画　公開後", filename: "testdata/youtube/video-after.json", SongVideoCount: 0, maybeSongVideoCount: 0},
		{Name: "歌動画　判断不可", filename: "testdata/youtube/video-maybe.json", SongVideoCount: 0, maybeSongVideoCount: 1},
		{Name: "生放送　公開前", filename: "testdata/youtube/stream-before.json", SongVideoCount: 0, maybeSongVideoCount: 0},
		{Name: "生放送　公開中", filename: "testdata/youtube/stream-now.json", SongVideoCount: 0, maybeSongVideoCount: 0},
		{Name: "生放送　公開後", filename: "testdata/youtube/stream-after.json", SongVideoCount: 0, maybeSongVideoCount: 0},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var res youtube.VideoListResponse
			data, err := os.ReadFile(test.filename)
			if err != nil {
				t.Error(err)
			}
			if err := json.Unmarshal(data, &res); err != nil {
				t.Error(err)
			}

			var videos []youtube.Video
			for _, v := range res.Items {
				videos = append(videos, *v)
			}

			songVideos, maybeSongVideos := CheckSong(videos)

			if len(songVideos) != test.SongVideoCount {
				t.Errorf("Expected %d songs, got %d", test.SongVideoCount, len(songVideos))
			}

			if len(maybeSongVideos) != test.maybeSongVideoCount {
				t.Errorf("Expected %d songs, got %d", test.maybeSongVideoCount, len(maybeSongVideos))
			}
		})
	}
}

func TestSendMaybeSongVideosForDiscord(t *testing.T) {
	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Error(err)
	}
	vid := "xMTOI62ws2I"
	videos, err := yt.Videos([]string{vid})
	if err != nil {
		t.Error(err)
	}

	err = SendMaybeSongVideosForDiscord(videos)
	if err != nil {
		t.Error(err)
	}
}
