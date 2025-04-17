package task

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/avast/retry-go/v4"
	"google.golang.org/api/youtube/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Task struct {
	Client     *cloudtasks.Client
	projectID  string
	locationID string
}

type TaskInfo struct {
	Video      youtube.Video
	QueueID    string
	URL        string
	MinutesAgo time.Duration
}

func NewTask() (*Task, error) {
	ctx := context.Background()
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing cloudtasks client: %v", err)
	}

	return &Task{
		client,
		os.Getenv("PROJECT_ID"),
		os.Getenv("LOCATION_ID"),
	}, nil
}

// 動画開始時刻の 〇分前 に指定のURLにHTTPリクエストを送るタスクを作成
// 指定されたURLには 動画ID が付属される
func (t *Task) Create(info *TaskInfo) error {
	ctx := context.Background()

	// 実行時刻より過去の時間を指定すると、すぐにタスクが実行される
	v := info.Video
	scheduleTime := time.Now()
	if v.LiveStreamingDetails != nil {
		vstime, _ := time.Parse("2006-01-02T15:04:05Z", v.LiveStreamingDetails.ScheduledStartTime)
		scheduleTime = vstime.Add(-info.MinutesAgo)

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
		Parent: fmt.Sprintf("projects/%s/locations/%s/queues/%s", t.projectID, t.locationID, info.QueueID),
		Task: &taskspb.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        fmt.Sprintf("%s?v=%s", info.URL, v.Id),
				},
			},
			ScheduleTime: timestamppb.New(scheduleTime),
		},
	}

	slog.Info("CreateTask",
		slog.String("video_id", v.Id),
		slog.String("video_title", v.Snippet.Title),
	)

	err := retry.Do(
		func() error {
			_, err := t.Client.CreateTask(ctx, req)
			// 既に登録済みのタスクの場合、警告ログを表示　エラーは返さない
			if err != nil && strings.Contains(err.Error(), "AlreadyExists") {
				slog.Warn(err.Error(),
					slog.String("video_id", v.Id),
				)
				return nil
			}
			return err
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
	)
	if err != nil {
		slog.Error(err.Error(),
			slog.String("video_id", v.Id),
		)
		return err
	}

	return nil
}
