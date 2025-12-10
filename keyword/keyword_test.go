package main

import (
	"log"
	"testing"

	"github.com/joho/godotenv"
)

func TestNotifyKeywordOnDiscord(t *testing.T) {
	err := godotenv.Load(".env.dev")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	vid := "9wZRPDWJ228"
	err = NotifyKeywordOnDiscord(vid)
	if err != nil {
		t.Fatal(err)
	}
}
