package fcm

import (
	"testing"
)

func TestSend(t *testing.T) {
	fcm := NewFCM()

	err := fcm.Notification(
		"5分後に公開",
		[]string{},
		&NotificationVideo{
			ID:        "Sqpmvv8uulM",
			Title:     "心予報/歌わせていただきました。",
			Thumbnail: "https://i.ytimg.com/vi/OPzbUoLxYyE/default.jpg",
		},
	)
	if err != nil {
		t.Error(err)
	}
}
