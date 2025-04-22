package newvideo

import (
	"fmt"
	"os"
	"testing"

	"github.com/aopontann/niji-tuu/internal/common/db"
	"github.com/aopontann/niji-tuu/internal/common/youtube"
)

func TestCheckNewVideoJob(t *testing.T) {
	// 新しく動画をアップロードしたプレイリスト情報を取得
	err := CheckNewVideoJob()
	if err != nil {
		t.Error(err)
	}
}

func TestNewVideoWebHook(t *testing.T) {
	vids := []string{"test1", "teset2"}
	err := NewVideoWebHook(vids)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStatusChengedVtubers(t *testing.T) {
	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Error(err)
	}
	db, err := db.NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	vtuber, err := GetStatusChengedVtubers(yt, db)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(vtuber)
}
