// Command createHTTPtask constructs and adds a task to a Cloud Tasks Queue.
package nsa

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewTask() *Task {
	ctx := context.Background()
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	return &Task{client}
}

func (t *Task) CreateSongTask(v youtube.Video) error {
	projectID := os.Getenv("PROJECT_ID")
	locationID := os.Getenv("LOCATION_ID")
	queueID := os.Getenv("QUEUE_ID")
	url := os.Getenv("DEMO_URL")
	ctx := context.Background()

	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)

	var scheduleTime time.Time
	if v.LiveStreamingDetails != nil {
		vstime, _ := time.Parse("2006-01-02T15:04:05Z", v.LiveStreamingDetails.ScheduledStartTime)
		vstime5mAgo := vstime.Add(-time.Minute * 5)

		if vstime5mAgo.After(time.Now()) && time.Until(vstime5mAgo).Minutes() > 5 {
			scheduleTime = vstime5mAgo
		} else {
			scheduleTime = time.Now()
		}
	} else {
		scheduleTime = time.Now()
	}

	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#HttpRequest
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
		if strings.Contains(err.Error(), "AlreadyExists") {
			slog.Warn("CreateSongTask",
				slog.String("severity", "WARNING"),
				slog.String("video_id", v.Id),
				slog.String("error_message", err.Error()),
			)
			return nil
		} else {
			slog.Error("CreateSongTask",
				slog.String("severity", "ERROR"),
				slog.String("video_id", v.Id),
				slog.String("error_message", err.Error()),
			)
			return err
		}
	}

	return nil
}

func (t *Task) CreateTopicTask(v youtube.Video, name string) error {
	projectID := os.Getenv("PROJECT_ID")
	locationID := os.Getenv("LOCATION_ID")
	queueID := os.Getenv("QUEUE_ID")
	url := os.Getenv("DEMO_URL")
	ctx := context.Background()

	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)

	var scheduleTime time.Time
	if v.LiveStreamingDetails != nil {
		vstime, _ := time.Parse("2006-01-02T15:04:05Z", v.LiveStreamingDetails.ScheduledStartTime)
		vstime1hAgo := vstime.Add(-time.Hour)

		if vstime1hAgo.After(time.Now()) && time.Until(vstime1hAgo).Minutes() > 60 {
			scheduleTime = vstime1hAgo
		} else {
			scheduleTime = time.Now()
		}
	} else {
		scheduleTime = time.Now()
	}

	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#HttpRequest
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

	body := &TopicTaskReqBody{ID: v.Id, Name: name}
	j, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req.Task.GetHttpRequest().Body = j

	_, err = t.Client.CreateTask(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "AlreadyExists") {
			slog.Warn("CreateTopicTask",
				slog.String("severity", "WARNING"),
				slog.String("video_id", v.Id),
				slog.String("error_message", err.Error()),
			)
			return err
		} else {
			slog.Error("CreateTopicTask",
				slog.String("severity", "ERROR"),
				slog.String("video_id", v.Id),
				slog.String("error_message", err.Error()),
			)
			return err
		}
	}

	return nil
}
