package nsa

import (
	"testing"

	"github.com/joho/godotenv"
)

func TestCheckNewVideoJob(t *testing.T) {
	godotenv.Load(".env")
	// 新しく動画をアップロードしたプレイリスト情報を取得
	err := CheckNewVideoJob()
	if err != nil {
		t.Error(err)
	}
}

func TestNewVideoWebHook(t *testing.T) {
	godotenv.Load(".env.test")

	vids := []string{"test1", "teset2"}
	err := NewVideoWebHook(vids)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateTaskToNoficationByDiscord(t *testing.T) {
	godotenv.Load(".env.test")

	vids := []string{"EgaXyUcsM48"}
	err := CreateTaskToNoficationByDiscord(vids)
	if err != nil {
		t.Error(err)
	}
}

func TestDiscordAnnounceJob(t *testing.T) {
	godotenv.Load(".env")
	// 新しく動画をアップロードしたプレイリスト情報を取得
	err := DiscordAnnounceJob("cOaucoqw1Rs")
	if err != nil {
		t.Error(err)
	}
}
