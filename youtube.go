package nsa

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"

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
func (y *Youtube) Playlists(plist []string) (map[string]int64, error) {
	newlist := make(map[string]int64, 500)
	for i := 0; i*50 <= len(plist); i++ {
		var id string
		if len(plist) > 50*(i+1) {
			id = strings.Join(plist[50*i:50*(i+1)], ",")
		} else {
			id = strings.Join(plist[50*i:], ",")
		}
		call := y.Service.Playlists.List([]string{"snippet", "contentDetails"}).MaxResults(50).Id(id)
		res, err := call.Do()
		if err != nil {
			slog.Error("playlists-list",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
		}

		for _, item := range res.Items {
			newlist[item.Id] = item.ContentDetails.ItemCount
			slog.Debug("youtube-playlists-list",
				slog.String("severity", "DEBUG"),
				slog.String("PlaylistId", item.Id),
				slog.Int64("ItemCount", item.ContentDetails.ItemCount),
			)
		}
	}
	return newlist, nil
}

func (y *Youtube) PlaylistItems(plist []string) ([]string, error) {
	// 動画IDを格納する文字列型配列を宣言
	vidList := make([]string, 0, 1500)

	for _, pid := range plist {
		var rid []string
		call := y.Service.PlaylistItems.List([]string{"snippet"}).PlaylistId(pid).MaxResults(3)
		res, err := call.Do()
		if err != nil {
			slog.Error("playlistitems-list",
				slog.String("severity", "ERROR"),
				slog.String("message", err.Error()),
			)
			return []string{}, err
		}

		for _, item := range res.Items {
			rid = append(rid, item.Snippet.ResourceId.VideoId)
			vidList = append(vidList, item.Snippet.ResourceId.VideoId)
		}

		slog.Debug("youtube-playlistitems-list",
			slog.String("severity", "DEBUG"),
			slog.String("PlaylistId", pid),
			slog.String("videoId", strings.Join(rid, ",")),
		)
	}
	return vidList, nil
}

// RSSから過去30分間にアップロードされた動画IDを取得
func (y *Youtube) RssFeed(pids []string) ([]string, error) {
	var vids []string
	for _, pid := range pids {
		resp, err := http.Get("https://www.youtube.com/feeds/videos.xml?playlist_id=" + pid)
		if err != nil {
			slog.Error("youtube-rss",
				slog.String("severity", "ERROR"),
				slog.String("playlist_id", pid),
				slog.String("error_message", err.Error()),
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

// 公開前、公開中の動画IDを取得
func (y *Youtube) UpcomingLiveVideoIDs(pids []string) ([]string, error) {
	// 公開前、公開中の動画IDリスト
	var resVIDs []string

	vidPattern := `"videoId":".{11}"`
	stylePattern := `"style":"(UPCOMING|LIVE|DEFAULT)"`
	// 正規表現をコンパイル
	vidReg, err := regexp.Compile(vidPattern)
	if err != nil {
		return nil, err
	}
	styleReg, err := regexp.Compile(stylePattern)
	if err != nil {
		return nil, err
	}

	for _, pid := range pids {
		body, err := retry.DoWithData(
			// 500が返ってきた場合もリトライ
			func() ([]byte, error) {
				resp, err := http.Get("https://www.youtube.com/playlist?list=" + pid)
				if err != nil {
					resp.Body.Close()
					return nil, err
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}

				if resp.StatusCode != http.StatusOK {
					slog.Warn("UpcomingLiveVideoId",
						slog.String("severity", "WARNING"),
						slog.String("playlist_id", pid),
						slog.Int("status_code", resp.StatusCode),
						slog.String("text", string(body)),
					)
				}

				// ステータスコード500が返ってきた場合
				if resp.StatusCode == http.StatusInternalServerError {
					return nil, errors.New("status_code:500")
				}

				return body, nil
			},
			retry.Attempts(3),
			retry.Delay(1*time.Second),
		)
		if err != nil {
			return nil, err
		}
		text := string(body)

		for _, t := range strings.Split(text, "playlistVideoRenderer")[1:] {
			strVID := vidReg.FindString(t)
			strStype := styleReg.FindString(t)
			if strStype == "" {
				continue
			}
			vid := strings.Split(strVID, ":")[1]
			style := strings.Split(strStype, ":")[1]

			if style[1:len(style)-1] != "DEFAULT" {
				resVIDs = append(resVIDs, vid[1:len(vid)-1])
			}
		}
	}
	return resVIDs, nil
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

// 5分以内に公開される動画か
func (y *Youtube) IsStartWithin5m(video youtube.Video) bool {
	// プレミア公開、生放送終了した動画
	if video.LiveStreamingDetails == nil || video.Snippet.LiveBroadcastContent == "none" {
		return false
	}

	sst, _ := time.Parse("2006-01-02T15:04:05Z", video.LiveStreamingDetails.ScheduledStartTime)
	sub := sst.Sub(time.Now().UTC()).Minutes()

	return sub < 5 && sub >= 0
}

// 歌ってみた動画のタイトルによく含まれるキーワードが 指定した動画に含まれているか
func (y *Youtube) FindSongKeyword(video youtube.Video) bool {
	for _, word := range getSongWordList() {
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

// 消されていない動画か
func (y *Youtube) IsExistsVideo(vid string) (bool, error) {
	call := y.Service.Videos.List([]string{"id"}).Id(vid).MaxResults(1)
	res, err := call.Do()
	if err != nil {
		return false, err
	}
	if len(res.Items) == 0 {
		return false, nil
	}
	return true, nil
}
