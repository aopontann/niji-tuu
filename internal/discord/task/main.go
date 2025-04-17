package task

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aopontann/niji-tuu/internal/common/task"
	"github.com/aopontann/niji-tuu/internal/common/youtube"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	vids := r.FormValue("v")
	if vids == "" {
		msg := "クエリパラメータ v が指定されていません"
		slog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	err := CreateTaskToNoficationByDiscord(strings.Split(vids, ","))
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 受け取った動画IDから動画の公開予定時刻を取得し、公開1時間前に通知するタスクを登録
func CreateTaskToNoficationByDiscord(vids []string) error {
	slog.Info("処理開始",
		slog.String("vids", strings.Join(vids, ",")),
	)
	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	ctask, err := task.NewTask()
	if err != nil {
		return err
	}

	videos, err := yt.Videos(vids)
	if err != nil {
		return err
	}

	// discord から通知するタスクを登録
	for _, v := range videos {
		err = ctask.Create(&task.TaskInfo{
			Video:      v,
			QueueID:    os.Getenv("DISCORD_QUEUE_ID"),
			URL:        os.Getenv("DISCORD_URL"),
			MinutesAgo: time.Hour,
		})
		if err != nil {
			return err
		}
	}
	slog.Info("処理終了")
	return nil
}
