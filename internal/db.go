package internal

import (
	"crypto/tls"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

//func init() {
//	var err error
//	if os.Getenv("ENV") != "prod" {
//		dsn := os.Getenv("DSN")
//		db, err = sqlx.Connect("mysql", dsn)
//	} else {
//		mysql.RegisterTLSConfig("register-tidb-tls", &tls.Config{
//			MinVersion: tls.VersionTLS12,
//			ServerName: os.Getenv("DB_HOST"),
//		})
//		dsn := os.Getenv("DSN") + "&tls=register-tidb-tls"
//		db, err = sqlx.Connect("mysql", dsn)
//	}
//	if err != nil {
//		panic(err)
//	}
//}

func NewDB(dsn string) *sqlx.DB {
	if os.Getenv("ENV") != "prod" {
		return sqlx.MustConnect("mysql", dsn)
	} else {
		mysql.RegisterTLSConfig("register-tidb-tls", &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: os.Getenv("DB_HOST"),
		})
		dsn = dsn + "&tls=register-tidb-tls"
		return sqlx.MustConnect("mysql", dsn)
	}
}

type Category struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Keyword struct {
	RoleID        string    `db:"role_id"`
	Name          string    `db:"name"`
	CategoryID    string    `db:"category_id"`
	ChannelID     string    `db:"channel_id"`
	InclusionList string    `db:"inclusion_list"`
	ExclusionList string    `db:"exclusion_list"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type User struct {
	Token     string    `db:"token"`
	Song      bool      `db:"song"`
	Info      bool      `db:"info"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Video struct {
	ID                 string    `db:"id"`
	Title              string    `db:"title"`
	Duration           string    `db:"duration"`
	Content            string    `db:"content"`
	ScheduledStartTime time.Time `db:"scheduled_start_time"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

type Vtuber struct {
	ID                string    `db:"id"`
	Name              string    `db:"name"`
	ItemCount         int64     `db:"item_count"`
	PlaylistLatestUrl string    `db:"playlist_latest_url"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

type Tables struct {
	Vtubers []Vtuber
	Videos  []Video
}

const (
	InsertKeywordsQuery = "INSERT INTO keywords (role_id, name, category_id, channel_id, inclusion_list) VALUES (:role_id, :name, :category_id, :channel_id, :inclusion_list)"
	InsertVideosQuery   = "INSERT IGNORE videos (id, title, duration, content, scheduled_start_time) VALUES (:id, :title, :duration, :content, :scheduled_start_time)"
	UpdateVtubersQuery  = "UPDATE vtubers SET item_count = :item_count, playlist_latest_url = :playlist_latest_url WHERE id = :id"
)

// テストで使用
const (
	InsertVtubersQuery = "INSERT INTO vtubers (id, name, item_count, playlist_latest_url) VALUES (:id, :name, :item_count, :playlist_latest_url)"
)

func SetUp(t Tables) error {
	if len(t.Vtubers) != 0 {
		_, err := db.NamedExec(InsertVtubersQuery, t.Vtubers)
		if err != nil {
			return err
		}
	}
	if len(t.Videos) != 0 {
		_, err := db.NamedExec(InsertVideosQuery, t.Videos)
		if err != nil {
			return err
		}
	}
	return nil
}

func CleanUp() error {
	_, err := db.Exec("TRUNCATE TABLE vtubers")
	if err != nil {
		return err
	}
	_, err = db.Exec("TRUNCATE TABLE videos")
	if err != nil {
		return err
	}
	return err
}
