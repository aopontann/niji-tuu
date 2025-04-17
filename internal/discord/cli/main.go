package discordcli

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func ListCommand() {
	godotenv.Load(".env.dev")
	guildID := os.Getenv("DISCORD_GUILD_ID")
	appID := os.Getenv("DISCORD_APP_ID")

	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		panic(err)
	}

	cmds, err := discord.ApplicationCommands(appID, guildID)
	if err != nil {
		panic(err)
	}
	for _, cmd := range cmds {
		fmt.Println(cmd.ID, cmd.Name)
	}
}

func BulkCommand() {
	godotenv.Load(".env.prod")
	guildID := os.Getenv("DISCORD_GUILD_ID")
	appID := os.Getenv("DISCORD_APP_ID")

	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		panic(err)
	}

	_, err = discord.ApplicationCommandBulkOverwrite(appID, guildID, []*discordgo.ApplicationCommand{
		{
			Name:        "song",
			Description: "歌みた動画の管理",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "歌みた動画の追加",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "url",
							Description: "動画のURL",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
		{
			Name:        "keyword",
			Description: "キーワードの管理",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "キーワードを登録する",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "keyword",
							Description: "登録するキーワード",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "category_id",
							Description: "追加先のカテゴリID",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
}
