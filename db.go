package nsa

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"google.golang.org/api/youtube/v3"
)

type Vtuber struct {
	bun.BaseModel `bun:"table:vtubers"`

	ID        string    `bun:"id,type:varchar(24),pk"`
	Name      string    `bun:"name,notnull,type:varchar"`
	ItemCount int64     `bun:"item_count,default:0,type:integer"`
	CreatedAt time.Time `bun:"created_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `bun:"updated_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
}

type Video struct {
	bun.BaseModel `bun:"table:videos"`

	ID        string    `bun:"id,type:varchar(11),pk"`
	Title     string    `bun:"title,notnull,type:varchar"`
	Duration  string    `bun:"duration,notnull,type:varchar"`
	Song      bool      `bun:"song,default:false,type:boolean"`
	Viewers   int64     `bun:"viewers,notnull,type:integer"`
	Content   string    `bun:"content,notnull,type:varchar"`
	StartTime time.Time `bun:"scheduled_start_time,type:timestamp"`
	Thumbnail string    `bun:"thumbnail,notnull,type:varchar"`
	CreatedAt time.Time `bun:"created_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `bun:"updated_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
}

type User struct {
	bun.BaseModel `bun:"table:users"`

	Token     string    `json:"token" bun:"token,type:varchar(1000),pk"`
	Song      bool      `json:"song" bun:"song,default:false,notnull,type:boolean"`
	Info      bool      `json:"info" bun:"info,default:false,notnull,type:boolean"`
	CreatedAt time.Time `bun:"created_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `bun:"updated_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
}

type Topic struct {
	bun.BaseModel `bun:"table:topics"`

	ID   string `bun:"id,type:int,pk"`
	Name string `bun:"name,type:varchar(100)"`
	// CreatedAt time.Time `bun:"created_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
	// UpdatedAt time.Time `bun:"updated_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
}

type UserTopic struct {
	bun.BaseModel `bun:"table:user_topics"`

	UserToken string    `bun:"user_token,type:varchar(1000),pk"`
	TopicID   string    `bun:"topic_id,type:varchar(20),pk"`
	CreatedAt time.Time `json:"created_at,omitempty" bun:"created_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `json:"updated_at,omitempty" bun:"updated_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
}

type DB struct {
	Service *bun.DB
}

func getSongWordList() []string {
	return []string{"cover", "歌って", "歌わせて", "Original Song", "オリジナル曲", "オリジナル楽曲", "オリジナルソング", "MV", "Music Video"}
}

func NewDB(db *bun.DB) *DB {
	return &DB{db}
}

func (db *DB) UpdatePlaylistItem(tx bun.Tx, newlist map[string]int64) error {
	ctx := context.Background()
	// DBを新しく取得したデータに更新
	var updateVideo []Vtuber
	for pid, v := range newlist {
		cid := strings.Replace(pid, "UU", "UC", 1)
		updateVideo = append(updateVideo, Vtuber{ID: cid, ItemCount: v, UpdatedAt: time.Now()})
	}

	if len(updateVideo) == 0 {
		return nil
	}

	_, err := tx.NewUpdate().Model(&updateVideo).Column("item_count", "updated_at").Bulk().Exec(ctx)
	if err != nil {
		slog.Error("update-itemCount",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return err
	}

	return nil
}

func (db *DB) PlaylistIDs() ([]string, error) {
	var cids []string
	ctx := context.Background()
	err := db.Service.NewSelect().Model((*Vtuber)(nil)).Column("id").Scan(ctx, &cids)
	if err != nil {
		return nil, err
	}

	var pids []string
	for _, cid := range cids {
		pids = append(pids, strings.Replace(cid, "UC", "UU", 1))
	}

	return pids, nil
}

// 動画情報をDBに登録　登録済みの動画は無視する
func (db *DB) SaveVideos(videos []youtube.Video) error {
	var Videos []Video
	for _, v := range videos {
		var Viewers int64
		Viewers = 0
		scheduledStartTime := "1998-01-01 15:04:05" // 例 2022-03-28T11:00:00Z
		if v.LiveStreamingDetails != nil {
			Viewers = int64(v.LiveStreamingDetails.ConcurrentViewers)
			// "2022-03-28 11:00:00"形式に変換
			rep1 := strings.Replace(v.LiveStreamingDetails.ScheduledStartTime, "T", " ", 1)
			scheduledStartTime = strings.Replace(rep1, "Z", "", 1)
		}
		t, _ := time.Parse("2006-01-02 15:04:05", scheduledStartTime)
		Videos = append(Videos, Video{
			ID:        v.Id,
			Title:     v.Snippet.Title,
			Duration:  v.ContentDetails.Duration,
			Content:   v.Snippet.LiveBroadcastContent,
			Viewers:   Viewers,
			Thumbnail: v.Snippet.Thumbnails.High.Url,
			StartTime: t,
			UpdatedAt: time.Now(),
		})
	}

	if len(Videos) == 0 {
		return nil
	}

	ctx := context.Background()
	_, err := db.Service.NewInsert().Model(&Videos).Ignore().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

// DBに登録されていない動画リストのみフィルター
func (db *DB) NotExistsVideos(videos []youtube.Video) ([]youtube.Video, error) {
	ctx := context.Background()
	// IN句に使用する動画IDリスト
	var sids []string
	for _, v := range videos {
		sids = append(sids, v.Id)
	}

	// 既に存在している動画IDリスト
	var ids []string
	err := db.Service.NewSelect().Model((*Video)(nil)).Column("id").Where("id IN (?)", bun.In(sids)).Scan(ctx, &ids)
	if err != nil {
		return nil, err
	}

	// 存在していない動画IDリスト
	var nids []string
	for _, sid := range sids {
		if !slices.Contains(ids, sid) {
			nids = append(nids, sid)
		}
	}

	// 存在していない動画情報リスト
	var nvideos []youtube.Video
	for _, v := range videos {
		if slices.Contains(nids, v.Id) {
			nvideos = append(nvideos, v)
		}
	}

	return nvideos, nil
}

// 5分後にプレミア公開される動画を取得
func (db *DB) songVideos5m() ([]Video, error) {
	ctx := context.Background()
	now, _ := time.Parse(time.RFC3339, time.Now().UTC().Format("2006-01-02T15:04:00Z"))
	tAfter := now.Add(1 * time.Second)
	tBefore := now.Add(5 * time.Minute)
	var videos []Video
	err := db.Service.NewSelect().
		Model(&videos).
		Where("? BETWEEN ? AND ?", bun.Ident("scheduled_start_time"), tAfter, tBefore).
		Where("duration != 'P0D'").
		Where("content = 'upcoming'").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	if len(videos) == 0 {
		return []Video{}, nil
	}

	var filtedVideos []Video

	for _, v := range videos {
		if v.Song {
			filtedVideos = append(filtedVideos, v)
			continue
		}
		for _, word := range getSongWordList() {
			if strings.Contains(strings.ToLower(v.Title), strings.ToLower(word)) {
				filtedVideos = append(filtedVideos, v)
				continue
			}
		}
	}

	return filtedVideos, nil
}

// songカラムがtrueのトークンリストを取得
func (db *DB) getSongTokens() ([]string, error) {
	// DBからチャンネルID、チャンネルごとの動画数を取得
	var tokens []string
	ctx := context.Background()
	err := db.Service.NewSelect().Model((*User)(nil)).Column("token").Where("song = true").Scan(ctx, &tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// ユーザーが登録しているキーワードのみを取得
func (db *DB) getTopicsUserRegister() ([]Topic, error) {
	ctx := context.Background()
	var topics []Topic
	_, err := db.Service.NewRaw(
		`SELECT id, name FROM topics
		WHERE EXISTS (SELECT 1 FROM user_topics WHERE topic_id = topics.id)`,
	).Exec(ctx, &topics)
	if err != nil {
		slog.Error("getTopics",
			slog.String("severity", "ERROR"),
			slog.String("message", err.Error()),
		)
		return nil, err
	}
	return topics, nil
}
