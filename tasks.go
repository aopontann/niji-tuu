package nsa

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/youtube/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Task struct {
	Client *cloudtasks.Client
}

type SongTaskReqBody struct {
	ID string `json:"id"`
}

type TopicTaskReqBody struct {
	VID string `json:"video_id"`
	TID int    `json:"topic_id"`
}

type ExistCheckTaskReqBody struct {
	ID string `json:"id"`
}

func NewTask() (*Task, error) {
	ctx := context.Background()
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing cloudtasks client: %v", err)
	}

	return &Task{client}, nil
}

// プレミア公開時刻の5分前にタスクが実行されるように、歌みたタスクを登録する
// 公開時刻が実行日より31日以上の場合、タスク登録はできないためエラーになる
func (t *Task) CreateSongTask(v youtube.Video) error {
	projectID := os.Getenv("PROJECT_ID")
	locationID := os.Getenv("LOCATION_ID")
	queueID := os.Getenv("SONG_QUEUE_ID")
	url := os.Getenv("SONG_URL")
	ctx := context.Background()

	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)

	// v.LiveStreamingDetails がある条件でタスクが作成されるため、v.LiveStreamingDetailsがあるかどうかのチェックは必要ない
	// 実行時刻より過去の時間を指定すると、すぐにタスクが実行される
	vstime, _ := time.Parse("2006-01-02T15:04:05Z", v.LiveStreamingDetails.ScheduledStartTime)
	scheduleTime := vstime.Add(-time.Minute * 5)

	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			Name: fmt.Sprintf("%s/tasks/%s", queuePath, v.Id),
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        url,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
			ScheduleTime: timestamppb.New(scheduleTime),
		},
	}

	body := &SongTaskReqBody{ID: v.Id}
	j, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req.Task.GetHttpRequest().Body = j

	_, err = t.Client.CreateTask(ctx, req)
	if err != nil {
		// 既に登録済みのタスクの場合、警告ログを表示　エラーは返さない
		if strings.Contains(err.Error(), "AlreadyExists") {
			slog.Warn(err.Error(),
				slog.String("video_id", v.Id),
			)
			return nil
			// 既に登録済みのエラー以外はエラーログを表示　エラーを返す
		} else {
			slog.Error(err.Error(),
				slog.String("video_id", v.Id),
			)
			return err
		}
	}

	return nil
}

// 放送開始時刻の1時間前に指定URLにHTTPリクエストが送られるタスクを登録
// 指定URLに動画IDのクエリパラメータを追加する（受け取り側はクエリパラメータから動画IDを取得）
// 公開時刻が1時間以内の場合、すぐにタスクを実行
// 公開時刻が実行日より31日以上の場合、タスク登録はできないためエラーになる
func (t *Task) CreateNewVideoTask(v youtube.Video, url string) error {
	projectID := os.Getenv("PROJECT_ID")
	locationID := os.Getenv("LOCATION_ID")
	queueID := os.Getenv("NEW_VIDEO_QUEUE_ID")
	ctx := context.Background()

	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)

	// 実行時刻より過去の時間を指定すると、すぐにタスクが実行される
	scheduleTime := time.Now()
	if v.LiveStreamingDetails != nil {
		vstime, _ := time.Parse("2006-01-02T15:04:05Z", v.LiveStreamingDetails.ScheduledStartTime)
		scheduleTime = vstime.Add(-time.Hour)

		// 31日以上の場合
		if time.Until(scheduleTime).Hours()/24 > 30 {
			slog.Warn("31日以降のタスクは登録できません",
				slog.String("video_id", v.Id),
				slog.String("video_title", v.Snippet.Title),
			)
			return nil
		}
	}

	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        fmt.Sprintf("%s?v=%s", url, v.Id),
				},
			},
			ScheduleTime: timestamppb.New(scheduleTime),
		},
	}

	slog.Info("CreateNewVideoTask",
		slog.String("video_id", v.Id),
		slog.String("video_title", v.Snippet.Title),
	)

	_, err := t.Client.CreateTask(ctx, req)
	if err != nil {
		slog.Error(err.Error(),
			slog.String("video_id", v.Id),
			slog.String("video_title", v.Snippet.Title),
		)
		return err
	}

	return nil
}
