package main

import (
	"os"

	nt "github.com/aopontann/niji-tuu"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func main() {
	godotenv.Load(".env.prod")
	config, err := pgx.ParseConfig(os.Getenv("DSN"))
	if err != nil {
		panic(err)
	}
	sqldb := stdlib.OpenDB(*config)
	db := bun.NewDB(sqldb, pgdialect.New())
	if err != nil {
		panic(err)
	}

	models := []interface{}{
		(*nt.Vtuber)(nil),
		(*nt.Video)(nil),
		(*nt.Role)(nil),
		(*nt.User)(nil),
	}

	data := modelsToByte(db, models)
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
