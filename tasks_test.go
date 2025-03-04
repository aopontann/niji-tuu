package nsa

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"google.golang.org/api/youtube/v3"
)

func TestCreateSongTask(t *testing.T) {
	// videos := loadTestVideos(t)
	godotenv.Load(".env")

	task, err := NewTask()
	if err != nil {
		t.Fatal(err)
	}

	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err)
	}

	videos, err := yt.Videos([]string{"gJMdSDGWEoM"})
	if err != nil {
		t.Fatal(err)
	}

	err = task.CreateSongTask(videos[0])
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewVideoTask(t *testing.T) {
	godotenv.Load(".env")
	videos := loadTestVideos(t)

	task, err := NewTask()
	if err != nil {
		t.Fatal(err)
	}

	url := "https://example.com"
	err = task.CreateNewVideoTask(*videos.Items[0], url)
	if err != nil {
		t.Error(err)
	}
}

func loadTestVideos(t *testing.T) *youtube.VideoListResponse {
	t.Helper()
	var videos youtube.VideoListResponse
	data, err := os.ReadFile("testdata/videos.json")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	if err := json.Unmarshal(data, &videos); err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}
	if len(videos.Items) == 0 {
		t.Fatal("Test data contains no video items")
	}
	return &videos
}
