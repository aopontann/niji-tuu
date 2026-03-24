package main

import (
	"testing"
	"time"

	"github.com/aopontann/niji-tuu/internal"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

//func TestFilterDiffVideos(t *testing.T) {
//	st2, _ := time.Parse("2006-01-02 15:04:05", "2027-11-30 14:45:00")
//	st3, _ := time.Parse("2006-01-02 15:04:05", "2024-05-02 02:00:00")
//	now := time.Now().UTC().Truncate(time.Second)
//	tests := []struct {
//		name          string
//		PrepareTables internal.Tables
//		vids          []string
//		Want          []internal.Video
//	}{
//		{
//			name: "正常",
//			PrepareTables: internal.Tables{
//				Videos: []internal.Video{
//					{ID: "--AselsUWdo", Title: "liveからnone", Duration: "P0D", Content: "live", ScheduledStartTime: st3},
//					{ID: "I0ZqZMGOn8U", Title: "更新前", Duration: "AAA", Content: "upcoming", ScheduledStartTime: now},
//				},
//			},
//			vids: []string{"--AselsUWdo", "I0ZqZMGOn8U"},
//			Want: []internal.Video{
//				{ID: "--AselsUWdo", Title: "Birthday Countdown Stream!", Duration: "PT2H44M27S", Content: "none", ScheduledStartTime: st3},
//				{ID: "I0ZqZMGOn8U", Title: "【予定表】るいす～どこにいるんだ～！【ルイス・キャミー】", Duration: "P0D", Content: "upcoming", ScheduledStartTime: st2},
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			if err := internal.CleanUp(db); err != nil {
//				t.Error(err)
//			}
//			if err := internal.SetUp(db, test.PrepareTables); err != nil {
//				t.Error(err)
//			}
//
//			yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
//			if err != nil {
//				t.Error(err)
//			}
//
//			videos, err := yt.Videos(test.vids)
//			if err != nil {
//				t.Error(err)
//			}
//
//			res, err := FilterDiffVideos(videos)
//			if err != nil {
//				t.Error(err)
//			}
//
//			t.Log(res)
//		})
//	}
//}

func TestUpdateVideoJob(t *testing.T) {
	st1, _ := time.Parse("2006-01-02 15:04:05", "2028-01-31 14:59:00")
	st2, _ := time.Parse("2006-01-02 15:04:05", "2027-11-30 14:45:00")
	st3, _ := time.Parse("2006-01-02 15:04:05", "2024-05-02 02:00:00")
	now := time.Now().UTC().Truncate(time.Second)

	opts := cmpopts.IgnoreFields(internal.Video{}, "CreatedAt", "UpdatedAt")

	tests := []struct {
		name          string
		PrepareTables internal.Tables
		Want          []internal.Video
	}{
		{
			name: "正常",
			PrepareTables: internal.Tables{
				Videos: []internal.Video{
					{ID: "--AselsUWdo", Title: "liveからnone", Duration: "P0D", Content: "live", ScheduledStartTime: st3},
					{ID: "I0ZqZMGOn8U", Title: "更新前", Duration: "AAA", Content: "upcoming", ScheduledStartTime: now},
					{ID: "LXtk9sF3A", Title: "title3", Duration: "PT5H39M24S", Content: "none", ScheduledStartTime: now},
					{ID: "kSCnJMV3I58", Title: "⭑𓂃. ⋆˙⟡𝔽𝕣𝕖𝕖 ℂ𝕙𝕒𝕥⁀ ⊹ ₊ “えま★おうがすと/にじさんじ所属", Duration: "P0D", Content: "upcoming", ScheduledStartTime: st1},
				},
			},
			Want: []internal.Video{
				{ID: "--AselsUWdo", Title: "Birthday Countdown Stream!", Duration: "PT2H44M27S", Content: "none", ScheduledStartTime: st3},
				{ID: "I0ZqZMGOn8U", Title: "【予定表】るいす～どこにいるんだ～！【ルイス・キャミー】", Duration: "P0D", Content: "upcoming", ScheduledStartTime: st2},
				{ID: "LXtk9sF3A", Title: "title3", Duration: "PT5H39M24S", Content: "none", ScheduledStartTime: now},
				{ID: "kSCnJMV3I58", Title: "⭑𓂃. ⋆˙⟡𝔽𝕣𝕖𝕖 ℂ𝕙𝕒𝕥⁀ ⊹ ₊ “えま★おうがすと/にじさんじ所属", Duration: "P0D", Content: "upcoming", ScheduledStartTime: st1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := internal.CleanUp(db); err != nil {
				t.Error(err)
			}
			if err := internal.SetUp(db, test.PrepareTables); err != nil {
				t.Error(err)
			}

			err := UpdateVideoJob()
			if err != nil {
				t.Error(err)
			}

			var resultVideos []internal.Video
			err = db.Select(&resultVideos, "SELECT * FROM videos")
			if err != nil {
				t.Error(err)
			}
			t.Log(resultVideos)
			if diff := cmp.Diff(test.Want, resultVideos, opts); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
