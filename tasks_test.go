package nsa

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/api/youtube/v3"
)

func TestCreateTask(t *testing.T) {
	// videos := loadTestVideos(t)
	godotenv.Load(".env")

	task, err := NewTask()
	if err != nil {
		t.Fatal(err)
	}

	videos := loadTestVideos(t)

	for _, v := range videos.Items {
		err = task.Create(&TaskInfo{
			Video:      *v,
			QueueID:    os.Getenv("DISCORD_QUEUE_ID"), //SONG_QUEUE_IDは指定しないように
			URL:        os.Getenv("DISCORD_URL"),
			MinutesAgo: time.Hour,
		})
		if err != nil {
			t.Error(err)
		}
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
