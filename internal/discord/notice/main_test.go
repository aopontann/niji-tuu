package discordnotice

import (
	"testing"

	"github.com/joho/godotenv"
)

func TestDiscordAnnounceJob(t *testing.T) {
	godotenv.Load(".env")
	// 新しく動画をアップロードしたプレイリスト情報を取得
	err := DiscordAnnounceJob("cOaucoqw1Rs")
	if err != nil {
		t.Error(err)
	}
}
