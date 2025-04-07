package nsa

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"google.golang.org/api/youtube/v3"
)

type Vtuber struct {
	bun.BaseModel `bun:"table:vtubers"`

	ID                string    `bun:"id,type:varchar(24),pk"`
	Name              string    `bun:"name,notnull,type:varchar"`
	ItemCount         int64     `bun:"item_count,default:0,type:integer"`
	PlaylistLatestUrl string    `bun:"playlist_latest_url,type:varchar,default:''"`
	CreatedAt         time.Time `bun:"created_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt         time.Time `bun:"updated_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
}

type Video struct {
	bun.BaseModel `bun:"table:videos"`

	ID        string    `bun:"id,type:varchar(11),pk"`
	Title     string    `bun:"title,notnull,type:varchar"`
	Duration  string    `bun:"duration,notnull,type:varchar"`
	Content   string    `bun:"content,notnull,type:varchar"`
	StartTime time.Time `bun:"scheduled_start_time,type:timestamp"`
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

type Role struct {
	bun.BaseModel `bun:"table:roles"`

	Name              string   `bun:"name,type:varchar(100),pk"`
	ID                string   `bun:"id,type:varchar(19),notnull"`
	ChannelID         string   `bun:"channel_id,type:varchar(30)"`
	Keywords          []string `bun:"keywords,array"`
	ExclusionKeywords []string `bun:"exclusion_keywords,array"`
	CreatedAt time.Time `bun:"created_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `bun:"updated_at,type:TIMESTAMP(0),nullzero,notnull,default:CURRENT_TIMESTAMP"`
}

type DB struct {
	Service *bun.DB
}

type Playlist struct {
	ItemCount int64
	Url       string
}

func NewDB(dsn string) (*DB, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	sqldb := stdlib.OpenDB(*config)
	db := bun.NewDB(sqldb, pgdialect.New())
	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.Service.Close()
}

// DBに登録されているPlaylistsの動画数を取得
func (db *DB) Playlists() (map[string]Playlist, error) {
	var vtubers []Vtuber
	ctx := context.Background()
	err := db.Service.NewSelect().Model(&vtubers).Scan(ctx)
	if err != nil {
		return nil, err
	}

	playlists := make(map[string]Playlist, 500)
	for _, vtuber := range vtubers {
		pid := strings.Replace(vtuber.ID, "UC", "UU", 1)
		playlists[pid] = Playlist{ItemCount: vtuber.ItemCount, Url: vtuber.PlaylistLatestUrl}
	}

	return playlists, nil
}

func (db *DB) UpdatePlaylistItem(playlists map[string]Playlist) error {
	ctx := context.Background()
	// DBを新しく取得したデータに更新
	var updateVtubers []Vtuber
	for pid, playlist := range playlists {
		cid := strings.Replace(pid, "UU", "UC", 1)
		updateVtubers = append(updateVtubers, Vtuber{ID: cid, ItemCount: playlist.ItemCount, PlaylistLatestUrl: playlist.Url, UpdatedAt: time.Now()})
	}

	if len(updateVtubers) == 0 {
		return nil
	}

	_, err := db.Service.NewUpdate().Model(&updateVtubers).Column("item_count", "playlist_latest_url", "updated_at").Bulk().Exec(ctx)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return nil
}

func (db *DB) UpdatePlaylistItemWithTx(tx bun.Tx, playlists map[string]Playlist) error {
	ctx := context.Background()
	// DBを新しく取得したデータに更新
	var updateVtubers []Vtuber
	for pid, playlist := range playlists {
		cid := strings.Replace(pid, "UU", "UC", 1)
		updateVtubers = append(updateVtubers, Vtuber{ID: cid, ItemCount: playlist.ItemCount, PlaylistLatestUrl: playlist.Url, UpdatedAt: time.Now()})
	}

	if len(updateVtubers) == 0 {
		return nil
	}

	_, err := tx.NewUpdate().Model(&updateVtubers).Column("item_count", "playlist_latest_url", "updated_at").Bulk().Exec(ctx)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return nil
}

func (db *DB) PlaylistIDs() ([]string, error) {
	var cids []string
	ctx := context.Background()
	err := db.Service.NewSelect().Model((*Vtuber)(nil)).Column("id").Scan(ctx, &cids)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	var pids []string
	for _, cid := range cids {
		pids = append(pids, strings.Replace(cid, "UC", "UU", 1))
	}

	return pids, nil
}

// 動画情報をDBに登録　登録済みの動画は無視する
func (db *DB) SaveVideos(tx bun.Tx, videos []youtube.Video) error {
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
func (db *DB) NotExistsVideoID(vids []string) ([]string, error) {
	ctx := context.Background()

	// 既に存在している動画IDリスト
	var ids []string
	err := db.Service.NewSelect().Model((*Video)(nil)).Column("id").Where("id IN (?)", bun.In(vids)).Scan(ctx, &ids)
	if err != nil {
		slog.Error(err.Error(),
			slog.String("vids", strings.Join(vids, ",")),
		)
		return nil, err
	}

	// 存在していない動画IDリスト
	var nids []string
	for _, vid := range vids {
		if !slices.Contains(ids, vid) {
			nids = append(nids, vid)
		}
	}

	return nids, nil
}

// songカラムがtrueのトークンリストを取得
func (db *DB) getSongTokens() ([]string, error) {
	// DBからチャンネルID、チャンネルごとの動画数を取得
	var tokens []string
	ctx := context.Background()
	err := db.Service.NewSelect().Model((*User)(nil)).Column("token").Where("song = true").Scan(ctx, &tokens)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	return tokens, nil
}

// discordのロールを取得
func (db *DB) GetRoles() ([]Role, error) {
	ctx := context.Background()
	var roles []Role
	err := db.Service.NewSelect().Model(&roles).Column("name", "id", "channel_id", "keywords", "exclusion_keywords").Scan(ctx)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return roles, nil
}
