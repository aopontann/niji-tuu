package nsa

import (
	"log/slog"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	discordbot "github.com/aopontann/niji-tuu/internal/discord/bot"
	discordnotice "github.com/aopontann/niji-tuu/internal/discord/notice"
	discordtask "github.com/aopontann/niji-tuu/internal/discord/task"
	newvideo "github.com/aopontann/niji-tuu/internal/new-video"
	songnotice "github.com/aopontann/niji-tuu/internal/song/notice"
	songtask "github.com/aopontann/niji-tuu/internal/song/task"
)

func init() {
	// Cloud Logging用のログ設定
	ops := slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				a.Key = "severity"
				level := a.Value.Any().(slog.Level)
				if level == slog.LevelWarn {
					a.Value = slog.StringValue("WARNING")
				}
			}

			return a
		},
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &ops))
	slog.SetDefault(logger)

	functions.HTTP("new-video", newvideo.Handler)

	functions.HTTP("song-task", songtask.Handler)

	functions.HTTP("discord-task", discordtask.Handler)

	functions.HTTP("song-notice", songnotice.Handler)

	functions.HTTP("discord-notice", discordnotice.Handler)

	functions.HTTP("discord-bot", discordbot.Handler)
}
