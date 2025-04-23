package main

import (
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/aopontann/niji-tuu/internal/common/db"
)

func main() {
	config, err := pgx.ParseConfig(os.Getenv("DSN"))
	if err != nil {
		panic(err)
	}
	sqldb := stdlib.OpenDB(*config)
	bundb := bun.NewDB(sqldb, pgdialect.New())

	models := []interface{}{
		(*db.Vtuber)(nil),
		(*db.Video)(nil),
		(*db.User)(nil),
		(*db.Keyword)(nil),
	}

	data := modelsToByte(bundb, models)
	os.WriteFile("schema.sql", data, 0777)
}

func modelsToByte(db *bun.DB, models []interface{}) []byte {
	var data []byte
	for _, model := range models {
		query := db.NewCreateTable().Model(model).WithForeignKeys()
		rawQuery, err := query.AppendQuery(db.Formatter(), nil)
		if err != nil {
			panic(err)
		}
		data = append(data, rawQuery...)
		data = append(data, ";\n"...)
	}
	return data
}
