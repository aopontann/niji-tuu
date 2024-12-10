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
