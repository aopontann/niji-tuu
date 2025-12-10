package main

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/aopontann/niji-tuu/internal"
	"github.com/google/go-cmp/cmp"
	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if os.Getenv("ENV") != "prod" {
		log.Println("Loading environmental variables...")
		if err := godotenv.Load("../.env.dev"); err != nil {
			log.Fatalln("failed to load env variables", err)
		}
	}
	db = internal.NewDB(os.Getenv("DSN"))
	m.Run()
}

func TestUpdateVtubers(t *testing.T) {
	tx := db.MustBegin()
	//var vtubers []Vtuber
	//err := db.Select(&vtubers, "select id, item_count, playlist_latest_url from niji_tuu.vtubers")
	vtubers := []internal.Vtuber{
		{ID: "UC-6rZgmxZSIbq786j3RD5ow", Name: "レオス・ヴィンセント", ItemCount: 976, PlaylistLatestUrl: "https://i.ytimg.com/vi/IaskGl4LDNk/hqdefault.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "UC0xry7czPasj1wPxR8L0MZg", Name: "早乙女ベリー", ItemCount: 556, PlaylistLatestUrl: "https://i.ytimg.com/vi/WHOGEY-Suzc/hqdefault_live.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	err := UpdateVtubers(vtubers, tx)
	if err != nil {
		t.Fatal(err)
	}
	tx.Commit()
}

func TestGetStatusChangedVtubers(t *testing.T) {
	tests := []struct {
		name          string
		PrepareTables internal.Tables
		Want          []internal.Vtuber
	}{
		{
			name: "正常",
			PrepareTables: internal.Tables{
				Vtubers: []internal.Vtuber{
					{ID: "UC-6rZgmxZSIbq786j3RD5ow", Name: "レオス・ヴィンセント", ItemCount: 973, PlaylistLatestUrl: "https://i.ytimg.com/vi/IaskGl4LDNk/hqdefault.jpg"},
					{ID: "UC0xry7czPasj1wPxR8L0MZg", Name: "早乙女ベリー", ItemCount: 555, PlaylistLatestUrl: "https://i.ytimg.com/vi/WHOGEY-Suzc/hqdefault_live.jpg"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := internal.CleanUp(); err != nil {
				t.Error(err)
			}
			if err := internal.SetUp(test.PrepareTables); err != nil {
				t.Error(err)
			}
			vtubers, err := GetStatusChangedVtubers()
			if err != nil {
				t.Error(err)
			}
			t.Log(vtubers)
		})
	}
}

func TestGetNewVideoIDs(t *testing.T) {
	tests := []struct {
		name          string
		Vtubers       []internal.Vtuber
		PrepareTables internal.Tables
		Want          []internal.Vtuber
	}{
		{
			name: "正常",
			Vtubers: []internal.Vtuber{
				{ID: "UC-6rZgmxZSIbq786j3RD5ow", Name: "レオス・ヴィンセント", ItemCount: 973, PlaylistLatestUrl: "https://i.ytimg.com/vi/IaskGl4LDNk/hqdefault.jpg"},
				{ID: "UC0xry7czPasj1wPxR8L0MZg", Name: "早乙女ベリー", ItemCount: 555, PlaylistLatestUrl: "https://i.ytimg.com/vi/WHOGEY-Suzc/hqdefault_live.jpg"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := internal.CleanUp(); err != nil {
				t.Error(err)
			}
			if err := internal.SetUp(test.PrepareTables); err != nil {
				t.Error(err)
			}

			vids, err := GetNewVideoIDs(test.Vtubers)
			if err != nil {
				t.Error(err)
			}
			t.Log(vids)
		})
	}
}

func TestGetNewVideoIDsWithRSS(t *testing.T) {
	tests := []struct {
		name          string
		PrepareTables internal.Tables
		Want          []internal.Vtuber
	}{
		{
			name: "正常",
			PrepareTables: internal.Tables{
				Vtubers: []internal.Vtuber{
					{ID: "UC-6rZgmxZSIbq786j3RD5ow", Name: "レオス・ヴィンセント", ItemCount: 973, PlaylistLatestUrl: "https://i.ytimg.com/vi/IaskGl4LDNk/hqdefault.jpg"},
					{ID: "UC0xry7czPasj1wPxR8L0MZg", Name: "早乙女ベリー", ItemCount: 555, PlaylistLatestUrl: "https://i.ytimg.com/vi/WHOGEY-Suzc/hqdefault_live.jpg"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := internal.CleanUp(); err != nil {
				t.Error(err)
			}
			if err := internal.SetUp(test.PrepareTables); err != nil {
				t.Error(err)
			}
			vids, err := GetNewVideoIDsWithRSS()
			if err != nil {
				t.Error(err)
			}
			t.Log(vids)
		})
	}
}

func TestFilterNotExistsVideoIDs(t *testing.T) {
	tests := []struct {
		name          string
		vids          []string
		PrepareTables internal.Tables
		Want          []string
	}{
		{
			name: "正常",
			vids: []string{"video1", "video4"},
			PrepareTables: internal.Tables{
				Videos: []internal.Video{
					{ID: "video1", Title: "title1", Duration: "PT5H39M24S", Content: "none", ScheduledStartTime: time.Now().UTC()},
					{ID: "video2", Title: "title2", Duration: "PT5H39M24S", Content: "none", ScheduledStartTime: time.Now().UTC()},
					{ID: "video3", Title: "title3", Duration: "PT5H39M24S", Content: "none", ScheduledStartTime: time.Now().UTC()},
				},
			},
			Want: []string{"video4"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := internal.CleanUp(); err != nil {
				t.Error(err)
			}
			if err := internal.SetUp(test.PrepareTables); err != nil {
				t.Error(err)
			}
			vids, err := FilterNotExistsVideoIDs(test.vids)
			if err != nil {
				t.Error(err)
			}
			t.Log(vids)
			if diff := cmp.Diff(test.Want, vids); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSaveVideos(t *testing.T) {
	tests := []struct {
		name          string
		vids          []string
		PrepareTables internal.Tables
	}{
		{
			name: "正常",
			vids: []string{"o4drd_kRVAs", "D_aIT77rLGc"},
			PrepareTables: internal.Tables{
				Videos: []internal.Video{
					{ID: "BzArAI_gm7Y", Title: "たかしの二次会", Duration: "P0D", Content: "upcoming", ScheduledStartTime: time.Now().UTC()},
				},
			},
		},
		{
			name: "重複",
			vids: []string{"BzArAI_gm7Y"},
			PrepareTables: internal.Tables{
				Videos: []internal.Video{
					{ID: "BzArAI_gm7Y", Title: "たかしの二次会", Duration: "P0D", Content: "upcoming", ScheduledStartTime: time.Now().UTC()},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := internal.CleanUp(); err != nil {
				t.Error(err)
			}
			if err := internal.SetUp(test.PrepareTables); err != nil {
				t.Error(err)
			}
			yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
			if err != nil {
				t.Error(err)
			}
			videos, err := yt.Videos(test.vids)
			if err != nil {
				t.Error(err)
			}

			tx := db.MustBegin()
			err = SaveVideos(videos, tx)
			if err != nil {
				t.Error(err)
			}

			err = tx.Commit()
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCheckNewVideoJob(t *testing.T) {
	t.Setenv("SONG_TASK_URL", "https://example.com/")
	t.Setenv("DISCORD_TASK_URL", "https://example.com/")

	// 実際のAPIを叩いていて時間によって取得データが変わるため、最新のデータに書き換えてテストをすること
	tests := []struct {
		name          string
		PrepareTables internal.Tables
		Want          []internal.Vtuber
	}{
		{
			name: "正常1",
			PrepareTables: internal.Tables{
				Vtubers: []internal.Vtuber{
					{ID: "UC-6rZgmxZSIbq786j3RD5ow", Name: "レオス・ヴィンセント", ItemCount: 972, PlaylistLatestUrl: "https://i.ytimg.com/vi/IaskGl4LDNk/hqdefault.jpg"},
					{ID: "UC0xry7czPasj1wPxR8L0MZg", Name: "早乙女ベリー", ItemCount: 555, PlaylistLatestUrl: "https://i.ytimg.com/vi/WHOGEY-Suzc/hqdefault_live.jpg"},
				},
			},
		},
		{
			name: "正常2",
			PrepareTables: internal.Tables{
				Vtubers: []internal.Vtuber{
					{ID: "UC-6rZgmxZSIbq786j3RD5ow", Name: "レオス・ヴィンセント", ItemCount: 972, PlaylistLatestUrl: "https://i.ytimg.com/vi/IaskGl4LDNk/hqdefault.jpg"},
					{ID: "UC0xry7czPasj1wPxR8L0MZg", Name: "早乙女ベリー", ItemCount: 555, PlaylistLatestUrl: "https://i.ytimg.com/vi/WHOGEY-Suzc/hqdefault_live.jpg"},
				},
				Videos: []internal.Video{
					{ID: "BPNvmIcfUFU", Title: "【ダンガンロンパ 】希望の学園と絶望の高校生を完全初見でプレイ！！#９【早乙女ベリー/にじさんじ】", Duration: "P0D", Content: "upcoming", ScheduledStartTime: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := internal.CleanUp(); err != nil {
				t.Error(err)
			}
			if err := internal.SetUp(test.PrepareTables); err != nil {
				t.Error(err)
			}
			err := CheckNewVideoJob()
			if err != nil {
				t.Error(err)
			}
		})
	}
}
