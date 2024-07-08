package nsa

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

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

func TestYoutubeDemo(t *testing.T) {
	youtubeApiKey := os.Getenv("YOUTUBE_API_KEY")
	yt := NewYoutube(youtubeApiKey)

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
			fmt.Println("OK!!!")
		}
	}
}

func TestPlaylists(t *testing.T) {
	youtubeApiKey := os.Getenv("YOUTUBE_API_KEY")
	yt := NewYoutube(youtubeApiKey)

	pidList := []string{"UUe_p3YEuYJb8Np0Ip9dk-FQ", "UUveZ9Ic1VtcXbsyaBgxPMvg"}
	playlists, err := yt.Playlists(pidList)
	if err != nil {
		t.Error(err)
	}

	if playlists["UUe_p3YEuYJb8Np0Ip9dk-FQ"] != 1 {
		t.Errorf("except 1, but %d", playlists["UUe_p3YEuYJb8Np0Ip9dk-FQ"])
	}

	if playlists["UUveZ9Ic1VtcXbsyaBgxPMvg"] != 18 {
		t.Errorf("except 18, but %d", playlists["UUveZ9Ic1VtcXbsyaBgxPMvg"])
	}
}

func TestPlaylistItems(t *testing.T) {
	youtubeApiKey := os.Getenv("YOUTUBE_API_KEY")
	yt := NewYoutube(youtubeApiKey)

	pidList := []string{"UUe_p3YEuYJb8Np0Ip9dk-FQ", "UUveZ9Ic1VtcXbsyaBgxPMvg"}
	vidList, err := yt.PlaylistItems(pidList)
	if err != nil {
		t.Error(err)
	}

	if len(vidList) != 4 {
		t.Errorf("except 4, but %d", len(vidList))
	}
}

func TestVideos(t *testing.T) {
	youtubeApiKey := os.Getenv("YOUTUBE_API_KEY")
	yt := NewYoutube(youtubeApiKey)

	vidList := []string{"o4Xhm5fVMBA", "jUdRrvEFZXc"}
	videos, err := yt.Videos(vidList)
	if err != nil {
		t.Error(err)
	}

	if len(videos) != 2 {
		t.Errorf("except 2, but %d", len(vidList))
	}
}

func TestIsStartWithin5m(t *testing.T) {
	// testdata/videos.json 75行目の時間を5分以内に手直しすること(UTC)
	youtubeApiKey := os.Getenv("YOUTUBE_API_KEY")
	yt := NewYoutube(youtubeApiKey)

	var res youtube.VideoListResponse
	data, err := os.ReadFile("testdata/videos.json")
	if err != nil {
		t.Error(err)
	}
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		t.Error(err)
	}

	// 目視確認用
	for _, v := range res.Items {
		if yt.IsStartWithin5m(*v) {
			fmt.Println("true: ", v.Id)
		} else {
			fmt.Println("false: ", v.Id)
		}
	}

	if !yt.IsStartWithin5m(*res.Items[0]) {
		t.Error("failed")
	}
}