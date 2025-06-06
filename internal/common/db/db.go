package db

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
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

type Keyword struct {
	bun.BaseModel `bun:"table:keywords"`

	Name      string    `bun:"name,type:varchar(100),pk"`
	RoleID    string    `bun:"role_id,type:varchar(19),notnull"`
	ChannelID string    `bun:"channel_id,type:varchar(30)"`
	Include   []string  `bun:"include,array"`
	Ignore    []string  `bun:"ignore,array"`
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

func (db *DB) GetVtubers() ([]Vtuber, error) {
	var vtubers []Vtuber
	ctx := context.Background()
	err := db.Service.NewSelect().Model(&vtubers).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return vtubers, nil
}

func (db *DB) UpdateVtubers(vtubers []Vtuber, tx *bun.Tx) error {
	ctx := context.Background()
	if len(vtubers) == 0 {
		return nil
	}

	return retry.Do(
		func() error {
			var err error
			if tx != nil {
				_, err = tx.NewUpdate().Model(&vtubers).Bulk().Exec(ctx)
			} else {
				_, err = db.Service.NewUpdate().Model(&vtubers).Bulk().Exec(ctx)
			}
			return err
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
	)
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
func (db *DB) SaveVideos(videos []youtube.Video, tx *bun.Tx) error {
	var Videos []Video
	for _, v := range videos {
		scheduledStartTime := "1998-01-01 15:04:05" // 例 2022-03-28T11:00:00Z
		if v.LiveStreamingDetails != nil {
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
			StartTime: t,
			UpdatedAt: time.Now(),
		})
	}

	if len(Videos) == 0 {
		return nil
	}

	ctx := context.Background()
	return retry.Do(
		func() error {
			var err error
			if tx != nil {
				_, err = tx.NewInsert().Model(&Videos).Ignore().Exec(ctx)
			} else {
				_, err = db.Service.NewInsert().Model(&Videos).Ignore().Exec(ctx)
			}
			return err
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
	)
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
func (db *DB) GetSongTokens() ([]string, error) {
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

func (db *DB) GetKeywords() ([]Keyword, error) {
	ctx := context.Background()
	var keywords []Keyword
	err := db.Service.NewSelect().Model(&keywords).Scan(ctx)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return keywords, nil
}