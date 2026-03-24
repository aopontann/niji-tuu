package main

import (
	"os"
	"time"

	"github.com/aopontann/niji-tuu/internal"
	"google.golang.org/api/youtube/v3"
)

// ConvertVideos youtube.Videoをinternal.videoに変換する関数
func ConvertVideos(videos []youtube.Video) []internal.Video {
	var res []internal.Video
	for _, v := range videos {
		var datetime string
		if v.LiveStreamingDetails != nil && v.LiveStreamingDetails.ScheduledStartTime != "" {
			datetime = v.LiveStreamingDetails.ScheduledStartTime
		} else if v.LiveStreamingDetails != nil && v.LiveStreamingDetails.ActualStartTime != "" {
			datetime = v.LiveStreamingDetails.ActualStartTime
		} else {
			datetime = v.Snippet.PublishedAt
		}

		t, _ := time.Parse("2006-01-02T15:04:05Z", datetime)
		cv := internal.Video{
			ID:                 v.Id,
			Title:              v.Snippet.Title,
			Duration:           v.ContentDetails.Duration,
			Content:            v.Snippet.LiveBroadcastContent,
			ScheduledStartTime: t,
		}
		res = append(res, cv)
	}
	return res
}

// DiffVideos FilterDiffVideos DBに登録されたデータと異なっているデータのみにフィルター
func DiffVideos(old, new []internal.Video) (res []internal.Video) {
	// oldを参照しやすいようにマップ化
	oldMap := make(map[string]internal.Video)
	for _, v := range old {
		oldMap[v.ID] = v
	}

	// 差分チェック
	for _, nv := range new {
		ov := oldMap[nv.ID]
		if ov.Title != nv.Title || ov.Duration != nv.Duration || ov.Content != nv.Content || ov.ScheduledStartTime != nv.ScheduledStartTime {
			res = append(res, nv)
		}
	}

	return res
}

func UpdateVideoJob() error {
	var oldVideos []internal.Video
	err := db.Select(&oldVideos, "SELECT * FROM videos WHERE content = 'upcoming' OR content = 'live'")
	if err != nil {
		return err
	}
	if len(oldVideos) == 0 {
		return nil
	}

	var vids []string
	for _, video := range oldVideos {
		vids = append(vids, video.ID)
	}
	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}

	nv, err := yt.Videos(vids)
	if err != nil {
		return err
	}
	if len(nv) == 0 {
		return nil
	}
	newVideos := ConvertVideos(nv)

	diffVideos := DiffVideos(oldVideos, newVideos)

	// 新しく取得した動画情報でDBを上書き
	tx := db.MustBegin()
	stmt, err := tx.PrepareNamed(internal.UpdateVideosQuery)
	if err != nil {
		return err
	}
	for _, v := range diffVideos {
		_, err := stmt.Exec(v)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
