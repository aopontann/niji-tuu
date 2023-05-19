// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0

package db

import (
	"database/sql"
	"time"
)

type Video struct {
	ID                 string
	Title              sql.NullString
	Songconfirm        sql.NullInt32
	ScheduledStartTime sql.NullTime
	TwitterID          sql.NullString
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Vtuber struct {
	ID        string
	Name      string
	ItemCount int32
	CreatedAt time.Time
	UpdatedAt time.Time
}
