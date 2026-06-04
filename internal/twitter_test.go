package internal

import (
	"os"
	"testing"
)

func TestTwitterNotification(t *testing.T) {
	twitter := NewTwitter(
		os.Getenv("TWITTER_CONSUMER_KEY"),
		os.Getenv("TWITTER_CONSUMER_SECRET"),
		os.Getenv("TWITTER_ACCESS_TOKEN"),
		os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
	)
	err := twitter.Notification(&NotificationVideo{
		ID:        "F0C5eWLLdlg",
		Title:     "Hello World",
		Thumbnail: "https://i.ytimg.com/vi/F0C5eWLLdlg/maxresdefault.jpg",
	})
	if err != nil {
		t.Error(err)
	}
}
