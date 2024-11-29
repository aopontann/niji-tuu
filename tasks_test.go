package nsa

import (
	"encoding/json"
	"os"
	"testing"

	"google.golang.org/api/youtube/v3"
)

func TestCreateSongTask(t *testing.T) {
	videos := loadTestVideos(t)

	task, err := NewTask()
	if err != nil {
		t.Fatal(err)
	}

	err = task.CreateSongTask(*videos.Items[0])
	if err != nil {
		t.Error(err)
	}
}

func TestCreateTopicTask(t *testing.T) {
	videos := loadTestVideos(t)

	task, err := NewTask()
	if err != nil {
		t.Fatal(err)
	}
	
	topic := &Topic{ID: 1, Name: "歌枠"}
	err = task.CreateTopicTask(*videos.Items[0], *topic)
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
