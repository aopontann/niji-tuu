package discordcli

import (
	"os"

	discordcli "github.com/aopontann/niji-tuu/internal/discord/cli"
)

// コマンドの登録を行うコマンドラインツール
func main() {
	mode := os.Args[1]
	if mode == "-lc" {
		discordcli.ListCommand()
	}
	if mode == "-bc" {
		discordcli.BulkCommand()
	}
}
