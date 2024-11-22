package nsa

import (
	"encoding/json"
	"os"
	"testing"

	"google.golang.org/api/youtube/v3"
)

func TestCreateSongTask(t *testing.T) {
	var videos youtube.VideoListResponse
	data, err := os.ReadFile("testdata/videos.json")
	if err != nil {
		t.Error(err)
	}
	if err := json.Unmarshal([]byte(data), &videos); err != nil {
		t.Error(err)
	}
	
	task := NewTask()
	err = task.CreateSongTask(*videos.Items[0])
	if err != nil {
		t.Error(err)
	}
}

func TestCreateTopicTask(t *testing.T) {
	var videos youtube.VideoListResponse
	data, err := os.ReadFile("testdata/videos.json")
	if err != nil {
		t.Error(err)
	}
	if err := json.Unmarshal([]byte(data), &videos); err != nil {
		t.Error(err)
	}
	
	task := NewTask()
	err = task.CreateTopicTask(*videos.Items[0], "歌枠")
	if err != nil {
		t.Error(err)
	}
}