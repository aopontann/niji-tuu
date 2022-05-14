package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var YoutubeService *youtube.Service

func main() {
	log.Print("starting server...")
	// .envの読み込み(開発環境の時のみ読み込むようにしたい)
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	ctx := context.Background()
	YoutubeService, err = youtube.NewService(ctx, option.WithAPIKey(os.Getenv("YOUTUBE_API_KEY")))
	if err != nil {
		log.Printf("Error youtube.NewService")
	}

	// DB接続初期化
	DBInit()

	h1 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "pong\n")
	}

	http.HandleFunc("/ping", h1)
	http.HandleFunc("/youtube", YoutubeHandler)
	http.HandleFunc("/twitter", TwitterHandler)

	log.Printf("listening on port %s", port)
	// Start HTTP server.
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
