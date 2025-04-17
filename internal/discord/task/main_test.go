package task

import (
	"testing"

	"github.com/joho/godotenv"
)

func TestCreateTaskToNoficationByDiscord(t *testing.T) {
	godotenv.Load(".env.test")

	vids := []string{"EgaXyUcsM48"}
	err := CreateTaskToNoficationByDiscord(vids)
	if err != nil {
		t.Error(err)
	}
}
