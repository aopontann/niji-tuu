package discordbot

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/aopontann/niji-tuu/internal/common/db"
	"github.com/aopontann/niji-tuu/internal/common/task"
	"github.com/aopontann/niji-tuu/internal/common/youtube"
)

type InteractionData struct {
	GuildID string `json:"guild_id"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Options []struct {
		Name    string `json:"name"`
		Options []struct {
			Name  string `json:"name"`
			Type  int    `json:"type"`
			Value string `json:"value"`
		} `json:"options"`
		Type int `json:"type"`
	} `json:"options"`
	Type int `json:"type"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	publicKey := os.Getenv("DISCORD_PUBLIC_KEY")
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		slog.Error("Error decoding hex string: " + err.Error())
		http.Error(w, "Error decoding hex string", http.StatusInternalServerError)
		return
	}

	if !discordgo.VerifyInteraction(r, publicKeyBytes) {
		slog.Error("Invalid request signature")
		http.Error(w, "invalid request signature", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error reading request body: " + err.Error())
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var interaction discordgo.Interaction
	if err := json.Unmarshal(body, &interaction); err != nil {
		http.Error(w, "Error unmarshalling request body", http.StatusInternalServerError)
		return
	}

	if interaction.Type == 1 {
		pongResp, err := json.Marshal(discordgo.InteractionResponse{
			Type: discordgo.InteractionResponsePong,
		})
		if err != nil {
			http.Error(w, "Error marshalling response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(pongResp)
		w.WriteHeader(http.StatusOK)
		return
	}

	if interaction.Type == 2 {
		var data InteractionData
		// discordgo.Interaction.Data に Name などのフィールドがないため、[]byteに変換して自作の構造体にマッピングする
		jsonData, err := json.Marshal(interaction.Data)
		if err != nil {
			http.Error(w, "Error marshalling request body", http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			http.Error(w, "Error unmarshalling request body", http.StatusInternalServerError)
			return
		}

		if data.Name == "song" {
			subCmdInfo := data.Options[0]

			if subCmdInfo.Name == "add" {
				url := subCmdInfo.Options[0].Value
				err := AddSong(url)

				if err != nil {
					SendMessage(w, "登録に失敗しました："+err.Error())
				} else {
					SendMessage(w, "登録しました")
				}
			}
		}
		if data.Name == "keyword" {
			subCmdInfo := data.Options[0]

			if subCmdInfo.Name == "add" {
				keyword := subCmdInfo.Options[0].Value
				categoryID := subCmdInfo.Options[1].Value
				err := AddKeyword(keyword, categoryID)

				if err != nil {
					SendMessage(w, "登録に失敗しました："+err.Error())
				} else {
					SendMessage(w, "登録しました")
				}
			}
		}
	}
}

func SendMessage(w http.ResponseWriter, content string) {
	response := discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	}

	resp, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
	w.WriteHeader(http.StatusOK)
}

func AddSong(url string) error {
	// urlが https://www.youtube.com/watch?v=C56ImfpThK0 の形式であるため、=で分割して2つ目の要素を取得
	vid := strings.Split(url, "=")[1]

	yt, err := youtube.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}

	ctask, err := task.NewTask()
	if err != nil {
		return err
	}

	videos, err := yt.Videos([]string{vid})
	if err != nil {
		return err
	}

	taskInfoFCM := &task.TaskInfo{
		Video:      videos[0],
		QueueID:    os.Getenv("SONG_QUEUE_ID"),
		URL:        os.Getenv("SONG_URL"),
		MinutesAgo: time.Minute * 5,
	}
	taskInfoDiscord := &task.TaskInfo{
		Video:      videos[0],
		QueueID:    os.Getenv("SONG_QUEUE_ID"),
		URL:        os.Getenv("SONG_DISCORD_URL"),
		MinutesAgo: time.Hour * 1,
	}

	if err := ctask.Create(taskInfoFCM); err != nil {
		slog.Error(err.Error())
		return err
	}
	if err := ctask.Create(taskInfoDiscord); err != nil {
		slog.Error(err.Error())
		return err
	}

	return nil
}

func AddKeyword(keyword string, categoryID string) error {
	guildID := os.Getenv("DISCORD_GUILD_ID")
	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		return err
	}

	channel, err := discord.GuildChannelCreate(guildID, keyword, discordgo.ChannelTypeGuildText)
	if err != nil {
		return err
	}

	_, err = discord.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{ParentID: categoryID})
	if err != nil {
		return err
	}

	role, err := discord.GuildRoleCreate(guildID, &discordgo.RoleParams{Name: keyword})
	if err != nil {
		return err
	}

	cdb, err := db.NewDB(os.Getenv("DSN"))
	if err != nil {
		return err
	}

	_, err = cdb.Service.NewInsert().Model(&db.Keyword{
		Name:      keyword,
		RoleID:    role.ID,
		ChannelID: channel.ID,
		Include:   []string{keyword},
	}).Exec(context.Background())
	if err != nil {
		return err
	}

	return nil
}
