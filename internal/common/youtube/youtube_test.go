package youtube

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"testing"

	"github.com/aopontann/niji-tuu/internal/common/db"
	"google.golang.org/api/youtube/v3"
)

func SetUp() {
	ops := slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				a.Key = "severity"
				level := a.Value.Any().(slog.Level)
				if level == slog.LevelWarn {
					a.Value = slog.StringValue("WARNING")
				}
			}

			return a
		},
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &ops))
	slog.SetDefault(logger)
}

func TestYoutubeDemo(t *testing.T) {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	var videos youtube.VideoListResponse
	data, err := os.ReadFile("testdata/videos.json")
	if err != nil {
		t.Error(err)
	}
	if err := json.Unmarshal([]byte(data), &videos); err != nil {
		t.Error(err)
	}

	call := yt.Service.Videos.List([]string{"snippet", "contentDetails", "liveStreamingDetails"}).Id("o4Xhm5fVMBA", "jUdRrvEFZXc").MaxResults(50)
	res, err := call.Do()
	if err != nil {
		t.Error(err)
	}

	for i, item := range res.Items {
		if reflect.DeepEqual(item, videos.Items[i]) {
			t.Log("OK!!!")
		}
	}
}

func TestRssFeed(t *testing.T) {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	vids, err := yt.RssFeed([]string{"UCC7rRD6P7RQcx0hKv9RQP4w"})
	if err != nil {
		t.Error(err)
	}
	t.Log("vids:", vids)
}

func TestVideos(t *testing.T) {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	vidList := []string{"o4Xhm5fVMBA", "jUdRrvEFZXc"}
	videos, err := yt.Videos(vidList)
	if err != nil {
		t.Error(err)
	}

	if len(videos) != 2 {
		t.Errorf("except 2, but %d", len(vidList))
	}
}

func TestFindSongKeyword(t *testing.T) {
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	var res youtube.VideoListResponse
	data, err := os.ReadFile("testdata/videos.json")
	if err != nil {
		t.Error(err)
	}
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		t.Error(err)
	}

	for _, v := range res.Items {
		if yt.FindSongKeyword(*v) {
			t.Log("TRUE:", v.Snippet.Title)
		} else {
			t.Log("FALSE:", v.Snippet.Title)
		}
	}
}

func TestRSSFeed(t *testing.T) {
	SetUp()
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	cdb, err := db.NewDB(os.Getenv("DSN"))
	if err != nil {
		t.Fatal(err.Error())
	}
	defer func(cdb *db.DB) {
		err := cdb.Close()
		if err != nil {
			t.Error(err)
		}
	}(cdb)

	pids, err := cdb.PlaylistIDs()
	if err != nil {
		t.Fatal(err.Error())
	}

	vids, err := yt.RssFeed(pids)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(vids)
}

func TestPlaylistItems(t *testing.T) {
	SetUp()
	yt, err := NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		t.Fatal(err.Error())
	}

	pids := []string{"UU0g1AE0DOjBYnLhkgoRWN1w", "abc"}
	vids, err := yt.PlaylistItems(pids)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(vids)
}
