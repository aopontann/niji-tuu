package nsa

import (
	"context"
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Youtube struct {
	Service *youtube.Service
}

type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Text    string   `xml:",chardata"`
	Yt      string   `xml:"yt,attr"`
	Media   string   `xml:"media,attr"`
	Xmlns   string   `xml:"xmlns,attr"`
	Link    []struct {
		Text string `xml:",chardata"`
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
	} `xml:"link"`
	ID        string `xml:"id"`
	ChannelId string `xml:"channelId"`
	Title     string `xml:"title"`
	Author    struct {
		Text string `xml:",chardata"`
		Name string `xml:"name"`
		URI  string `xml:"uri"`
	} `xml:"author"`
	Published string `xml:"published"`
	Entry     []struct {
		Text      string `xml:",chardata"`
		ID        string `xml:"id"`
		VideoId   string `xml:"videoId"`
		ChannelId string `xml:"channelId"`
		Title     string `xml:"title"`
		Link      struct {
			Text string `xml:",chardata"`
			Rel  string `xml:"rel,attr"`
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Author struct {
			Text string `xml:",chardata"`
			Name string `xml:"name"`
			URI  string `xml:"uri"`
		} `xml:"author"`
		Published string `xml:"published"`
		Updated   string `xml:"updated"`
		Group     struct {
			Text    string `xml:",chardata"`
			Title   string `xml:"title"`
			Content struct {
				Text   string `xml:",chardata"`
				URL    string `xml:"url,attr"`
				Type   string `xml:"type,attr"`
				Width  string `xml:"width,attr"`
				Height string `xml:"height,attr"`
			} `xml:"content"`
			Thumbnail struct {
				Text   string `xml:",chardata"`
				URL    string `xml:"url,attr"`
				Width  string `xml:"width,attr"`
				Height string `xml:"height,attr"`
			} `xml:"thumbnail"`
			Description string `xml:"description"`
			Community   struct {
				Text       string `xml:",chardata"`
				StarRating struct {
					Text    string `xml:",chardata"`
					Count   string `xml:"count,attr"`
					Average string `xml:"average,attr"`
					Min     string `xml:"min,attr"`
					Max     string `xml:"max,attr"`
				} `xml:"starRating"`
				Statistics struct {
					Text  string `xml:",chardata"`
					Views string `xml:"views,attr"`
				} `xml:"statistics"`
			} `xml:"community"`
		} `xml:"group"`
	} `xml:"entry"`
}

func NewYoutube(key string) (*Youtube, error) {
	ctx := context.Background()
	yt, err := youtube.NewService(ctx, option.WithAPIKey(key))
	if err != nil {
		return nil, err
	}
	return &Youtube{yt}, nil
}

// チャンネルIDをキー、プレイリストに含まれている動画数を値とした連想配列を返す
func (y *Youtube) Playlists(pids []string) (map[string]Playlist, error) {
	playlists := make(map[string]Playlist, 500)
	for i := 0; i*50 < len(pids); i++ {
		var id string
		if len(pids) > 50*(i+1) {
			id = strings.Join(pids[50*i:50*(i+1)], ",")
		} else {
			id = strings.Join(pids[50*i:], ",")
		}
		call := y.Service.Playlists.List([]string{"snippet", "contentDetails"}).MaxResults(50).Id(id)
		res, err := call.Do()
		if err != nil {
			slog.Error("Playlists",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			return nil, err
		}

		for _, item := range res.Items {
			playlists[item.Id] = Playlist{ItemCount: item.ContentDetails.ItemCount, Url: item.Snippet.Thumbnails.High.Url}
			slog.Debug("youtube-playlists-list",
				slog.String("severity", "DEBUG"),
				slog.String("PlaylistId", item.Id),
				slog.Int64("ItemCount", item.ContentDetails.ItemCount),
			)
		}
	}
	return playlists, nil
}

func (y *Youtube) PlaylistItems(pid string) ([]string, error) {
	// 動画IDを格納する文字列型配列を宣言
	vids := make([]string, 0, 10)

	call := y.Service.PlaylistItems.List([]string{"snippet"}).PlaylistId(pid).MaxResults(10)
	res, err := call.Do()
	if err != nil {
		slog.Error("PlaylistItems",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return []string{}, err
	}

	for i, item := range res.Items {
		vids[i] = item.Snippet.ResourceId.VideoId
	}

	return vids, nil
}

// RSSから過去30分間にアップロードされた動画IDを取得
func (y *Youtube) RssFeed(pids []string) ([]string, error) {
	var vids []string
	for _, pid := range pids {
		resp, err := http.Get("https://www.youtube.com/feeds/videos.xml?playlist_id=" + pid)
		if err != nil {
			slog.Error("RssFeed",
				slog.String("severity", "ERROR"),
				slog.String("playlist_id", pid),
				slog.String("message", err.Error()),
			)
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Warn("youtube-rss",
				slog.String("severity", "WARNING"),
				slog.String("playlist_id", pid),
				slog.Int("status_code", resp.StatusCode),
				slog.String("text", string(body)),
			)
			resp.Body.Close()
			continue
		}

		var feed Feed
		if err := xml.Unmarshal([]byte(body), &feed); err != nil {
			return nil, err
		}

		for _, entry := range feed.Entry {
			sst, _ := time.Parse("2006-01-02T15:04:05+00:00", entry.Published)
			if time.Now().UTC().Sub(sst).Minutes() <= 30 {
				slog.Debug("RssFeed",
					slog.String("severity", "DEBUG"),
					slog.String("id", entry.VideoId),
					slog.String("title", entry.Title),
					slog.String("published", entry.Published),
					slog.String("updated", entry.Updated),
				)
				vids = append(vids, entry.VideoId)
			}
		}
	}
	return vids, nil
}

// Youtube Data API から動画情報を取得
func (y *Youtube) Videos(vids []string) ([]youtube.Video, error) {
	var rlist []youtube.Video
	for i := 0; i*50 <= len(vids); i++ {
		var id string
		if len(vids) > 50*(i+1) {
			id = strings.Join(vids[50*i:50*(i+1)], ",")
		} else {
			id = strings.Join(vids[50*i:], ",")
		}
		call := y.Service.Videos.List([]string{"snippet", "contentDetails", "liveStreamingDetails"}).Id(id).MaxResults(50)
		res, err := call.Do()
		if err != nil {
			slog.Error("Videos",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			return nil, err
		}

		for _, video := range res.Items {
			scheduledStartTime := "" // 例 2022-03-28T11:00:00Z
			if video.LiveStreamingDetails != nil {
				// "2022-03-28 11:00:00"形式に変換
				rep1 := strings.Replace(video.LiveStreamingDetails.ScheduledStartTime, "T", " ", 1)
				scheduledStartTime = strings.Replace(rep1, "Z", "", 1)
			}

			rlist = append(rlist, *video)

			slog.Debug("youtube-video-list",
				slog.String("severity", "DEBUG"),
				slog.String("id", video.Id),
				slog.String("title", video.Snippet.Title),
				slog.String("duration", video.ContentDetails.Duration),
				slog.String("schedule", scheduledStartTime),
				slog.String("channel_id", video.Snippet.ChannelId),
			)
		}
	}
	return rlist, nil
}

// 歌ってみた動画のタイトルによく含まれるキーワードが 指定した動画に含まれているか
func (y *Youtube) FindSongKeyword(video youtube.Video) bool {
	songWords := []string{"cover", "歌って", "歌わせて", "Original Song", "オリジナル曲", "オリジナル楽曲", "オリジナルソング", "MV", "Music Video"}
	for _, word := range songWords {
		if strings.Contains(strings.ToLower(video.Snippet.Title), strings.ToLower(word)) {
			return true
		}
	}
	return false
}

// 無視するキーワードが 指定した動画に含まれているか
func (y *Youtube) FindIgnoreKeyword(video youtube.Video) bool {
	for _, word := range []string{"切り抜き", "ラジオ", "くろなん"} {
		if strings.Contains(strings.ToLower(video.Snippet.Title), strings.ToLower(word)) {
			return true
		}
	}
	return false
}
