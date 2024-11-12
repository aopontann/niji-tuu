package nsa

import (
	"testing"
)

const token = ""

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

func TestSetTopic(t *testing.T) {
	fcm := NewFCM()

	err := fcm.SetTopic(token, "歌枠")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteTopic(t *testing.T) {
	fcm := NewFCM()

	err := fcm.DeleteTopic(token, "歌枠")
	if err != nil {
		t.Error(err)
	}
}

func TestSendTopic(t *testing.T) {
	fcm := NewFCM()

	err := fcm.TopicNotification("歌枠", &NotificationVideo{
		ID:        "Sqpmvv8uulM",
		Title:     "心予報/歌わせていただきました。",
		Thumbnail: "https://i.ytimg.com/vi/OPzbUoLxYyE/default.jpg",
	})
	if err != nil {
		t.Error(err)
	}
}
