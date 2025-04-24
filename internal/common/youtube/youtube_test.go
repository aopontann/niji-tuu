package youtube

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aopontann/niji-tuu/internal/common/db"
	"google.golang.org/api/youtube/v3"
)

type VideosListResponse struct {
	Kind  string `json:"kind,omitempty"`
	Etag  string `json:"etag,omitempty"`
	Items []struct {
		Kind    string `json:"kind,omitempty"`
		Etag    string `json:"etag,omitempty"`
		ID      string `json:"id,omitempty"`
		Snippet struct {
			PublishedAt time.Time `json:"publishedAt,omitempty"`
			ChannelID   string    `json:"channelId,omitempty"`
			Title       string    `json:"title,omitempty"`
			Description string    `json:"description,omitempty"`
			Thumbnails  struct {
				Default struct {
					URL    string `json:"url,omitempty"`
					Width  int    `json:"width,omitempty"`
					Height int    `json:"height,omitempty"`
				} `json:"default,omitempty"`
				Medium struct {
					URL    string `json:"url,omitempty"`
					Width  int    `json:"width,omitempty"`
					Height int    `json:"height,omitempty"`
				} `json:"medium,omitempty"`
				High struct {
					URL    string `json:"url,omitempty"`
					Width  int    `json:"width,omitempty"`
					Height int    `json:"height,omitempty"`
				} `json:"high,omitempty"`
				Standard struct {
					URL    string `json:"url,omitempty"`
					Width  int    `json:"width,omitempty"`
					Height int    `json:"height,omitempty"`
				} `json:"standard,omitempty"`
				Maxres struct {
					URL    string `json:"url,omitempty"`
					Width  int    `json:"width,omitempty"`
					Height int    `json:"height,omitempty"`
				} `json:"maxres,omitempty"`
			} `json:"thumbnails,omitempty"`
			ChannelTitle         string   `json:"channelTitle,omitempty"`
			Tags                 []string `json:"tags,omitempty"`
			CategoryID           string   `json:"categoryId,omitempty"`
			LiveBroadcastContent string   `json:"liveBroadcastContent,omitempty"`
			Localized            struct {
				Title       string `json:"title,omitempty"`
				Description string `json:"description,omitempty"`
			} `json:"localized,omitempty"`
			DefaultAudioLanguage string `json:"defaultAudioLanguage,omitempty"`
		} `json:"snippet,omitempty"`
		ContentDetails struct {
			Duration        string `json:"duration,omitempty"`
			Dimension       string `json:"dimension,omitempty"`
			Definition      string `json:"definition,omitempty"`
			Caption         string `json:"caption,omitempty"`
			LicensedContent bool   `json:"licensedContent,omitempty"`
			ContentRating   struct {
			} `json:"contentRating,omitempty"`
			Projection string `json:"projection,omitempty"`
		} `json:"contentDetails,omitempty"`
		LiveStreamingDetails struct {
			ActualStartTime    time.Time `json:"actualStartTime,omitempty"`
			ActualEndTime      time.Time `json:"actualEndTime,omitempty"`
			ScheduledStartTime time.Time `json:"scheduledStartTime,omitempty"`
		} `json:"liveStreamingDetails,omitempty"`
	} `json:"items,omitempty"`
	PageInfo struct {
		TotalResults   int `json:"totalResults,omitempty"`
		ResultsPerPage int `json:"resultsPerPage,omitempty"`
	} `json:"pageInfo,omitempty"`
}

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
	defer cdb.Close()

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
