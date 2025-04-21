package songtask

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aopontann/niji-tuu/internal/common/task"
	"github.com/aopontann/niji-tuu/internal/common/youtube"
	"github.com/avast/retry-go/v4"
	multierror "github.com/hashicorp/go-multierror"
	yt "google.golang.org/api/youtube/v3"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	vids := r.FormValue("v")
		if vids == "" {
			msg := "クエリパラメータ v が指定されていません"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		err := SongVideoCheck(strings.Split(vids, ","))
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
}

// 受け取った動画IDが歌動画か解析し、歌動画だった場合はタスクを登録する
func SongVideoCheck(vids []string) error {
	slog.Info("処理開始",
		slog.String("vids", strings.Join(vids, ",")),
	)
	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}
	task, err := task.NewTask()
	if err != nil {
		return err
	}

	videos, err := yt.Videos(vids)
	if err != nil {
		return err
	}

	// エラー時にgoroutineの中断をしてほしくないのと、
	// 発生したエラーを全てハンドリングしたいので go-multierror を採用
	var meg multierror.Group

	meg.Go(func() error {
		err = retry.Do(
			func() error {
				return SendMailMaybeSongVideos(yt, videos)
			},
			retry.Attempts(3),
			retry.Delay(1*time.Second),
		)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
		return nil
	})

	meg.Go(func() error {
		err = retry.Do(
			func() error {
				return AddSongTaskToCloudTasks(yt, task, videos)
			},
			retry.Attempts(3),
			retry.Delay(1*time.Second),
		)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
		return nil
	})

	merr := meg.Wait()
	slog.Info("処理終了")
	return merr.ErrorOrNil()
}

// 歌みた動画か判別しづらい動画をメールに送信する
func SendMailMaybeSongVideos(yt *youtube.Youtube, videos []yt.Video) error {
	for _, v := range videos {
		if yt.FindSongKeyword(v) {
			continue
		}
		if v.LiveStreamingDetails == nil {
			continue
		}
		if v.Snippet.LiveBroadcastContent != "upcoming" {
			continue
		}
		if v.ContentDetails.Duration == "P0D" {
			continue
		}
		// 特定のキーワードを含んでいる場合
		if yt.FindIgnoreKeyword(v) {
			continue
		}

		body := []byte(fmt.Sprintf(`{"content": "https://www.youtube.com/watch?v=%s"}`, v.Id))
		resp, err := http.Post(
			os.Getenv("DISCORD_WEBHOOK_MAYBE_SONG"),
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}
	return nil
}

// cloud task に歌みた告知タスクを登録
func AddSongTaskToCloudTasks(yt *youtube.Youtube, ctask *task.Task, videos []yt.Video) error {
	for _, v := range videos {
		// 生放送ではない、プレミア公開されない動画の場合
		if v.LiveStreamingDetails == nil {
			continue
		}
		// 放送終了した場合
		if v.Snippet.LiveBroadcastContent == "none" {
			continue
		}
		// 生放送の場合
		if v.ContentDetails.Duration == "P0D" {
			continue
		}
		if !yt.FindSongKeyword(v) || yt.FindIgnoreKeyword(v) {
			continue
		}
		err := ctask.Create(&task.TaskInfo{
			Video:      v,
			QueueID:    os.Getenv("SONG_QUEUE_ID"),
			URL:        os.Getenv("SONG_URL"),
			MinutesAgo: time.Minute * 5,
		})
		if err != nil {
			return err
		}
		err = ctask.Create(&task.TaskInfo{
			Video:      v,
			QueueID:    os.Getenv("SONG_QUEUE_ID"),
			URL:        os.Getenv("SONG_DISCORD_URL"),
			MinutesAgo: time.Hour * 1,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
