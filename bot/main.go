package main

import (
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/amatsagu/tempest"
	"github.com/aopontann/niji-tuu/internal"
	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

var db *sqlx.DB
var s *discordgo.Session

type Category struct {
	ID   string
	Name string
}

const (
	SongVideoCategoryId = "1345291841528135720" // prod:1345291841528135720 dev:1349307959175282733
	SongVideoRoleId     = "1359103811339161701" // prod:1359103811339161701 dev:1447909356095144036
	successMessage      = "処理が完了しました。"
	errorMessage        = "予期しないエラーが発生しました。時間をおいて再度実施してください。"
)

func main() {
	if os.Getenv("ENV") != "prod" {
		log.Println("Loading environmental variables...")
		if err := godotenv.Load(".env.dev"); err != nil {
			log.Fatalln("failed to load env variables", err)
		}
	}

	db = internal.NewDB(os.Getenv("DSN"))
	var err error
	s, err = discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	log.Println("Creating new Tempest client...")
	client := tempest.NewClient(tempest.ClientOptions{
		Token:     os.Getenv("DISCORD_BOT_TOKEN"),
		PublicKey: os.Getenv("DISCORD_PUBLIC_KEY"),
	})

	testServerID, err := tempest.StringToSnowflake(os.Getenv("DISCORD_GUILD_ID")) // Register example commands only to this guild.
	if err != nil {
		log.Fatalln("failed to parse env variable to snowflake", err)
	}

	log.Println("Registering commands & static components...")

	// 歌動画告知登録
	client.RegisterCommand(songAddCommand)

	// キーワード新規追加
	client.RegisterCommand(keywordAddCommand)

	// キーワード登録・解除 関連
	client.RegisterCommand(buttonCommand)
	client.RegisterComponent([]string{"button-1", "button-2", "button-3"}, notificationManagementModal)
	client.RegisterModal("select_modal", selectedKeywordComponent)

	err = client.SyncCommandsWithDiscord([]tempest.Snowflake{testServerID}, nil, false)
	if err != nil {
		log.Fatalln("failed to sync local commands storage with Discord API", err)
	}

	http.HandleFunc("POST /v1205", client.DiscordRequestHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Serving application")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalln("something went terribly wrong", err)
	}
}

var buttonCommand = tempest.Command{
	Name:        "notify-manage-button",
	Description: "通知登録ボタンを表示",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		itx.SendReply(tempest.ResponseMessageData{
			Components: []tempest.LayoutComponent{
				tempest.ActionRowComponent{
					Type: tempest.ACTION_ROW_COMPONENT_TYPE,
					Components: []tempest.InteractiveComponent{
						tempest.ButtonComponent{
							Type:     tempest.BUTTON_COMPONENT_TYPE,
							CustomID: "button-1",
							Style:    tempest.SECONDARY_BUTTON_STYLE,
							Label:    "あ行～な行",
						},
						tempest.ButtonComponent{
							Type:     tempest.BUTTON_COMPONENT_TYPE,
							CustomID: "button-2",
							Style:    tempest.SECONDARY_BUTTON_STYLE,
							Label:    "は行～わ行",
						},
						tempest.ButtonComponent{
							Type:     tempest.BUTTON_COMPONENT_TYPE,
							CustomID: "button-3",
							Style:    tempest.SECONDARY_BUTTON_STYLE,
							Label:    "A～Z",
						},
					},
				},
			},
		}, false, nil)
	},
}

func notificationManagementModal(itx tempest.ComponentInteraction) {
	var offset int
	switch itx.Data.CustomID {
	case "button-1":
		offset = 0
	case "button-2":
		offset = 5
	case "button-3":
		offset = 10
	}
	// カテゴリ一覧を取得
	var categories []Category
	err := db.Select(&categories, "SELECT id, name FROM categories ORDER BY created_at LIMIT 5 OFFSET ?", offset)
	if err != nil {
		panic(err)
	}

	// ユーザに付与されているロールを取得
	var assignedRoleIDs []string
	for _, id := range itx.Member.RoleIDs {
		assignedRoleIDs = append(assignedRoleIDs, id.String())
	}
	log.Println("assignedRoleIDs: ", assignedRoleIDs)

	// ロールとカテゴリの紐づき情報をDBから取得
	var keywords []internal.Keyword
	err = db.Select(&keywords, "SELECT name, role_id, category_id FROM keywords")
	if err != nil {
		panic(err)
	}
	// 扱いやすいようにMAPに変換　キー：カテゴリID、値：ロールID,名前
	keywordsMap := make(map[string][]internal.Keyword)
	for _, k := range keywords {
		keywordsMap[k.CategoryID] = append(keywordsMap[k.CategoryID], internal.Keyword{RoleID: k.RoleID, Name: k.Name})
	}

	var layoutComponents []tempest.LayoutComponent
	for _, category := range categories {
		var menu []tempest.SelectMenuOption

		// 一つは必ずチェックを入れる必要があり、キーワードを一つも選択しない場合送信できないため、ダミーを一つ用意
		//Descriptionはモーダルでは使えないっぽい。一回使ってしまうと、バグ？でしばらくモーダルが使えなくなる。（モバイル版のみ）
		menu = append(menu, tempest.SelectMenuOption{
			Label: "選択なし", Value: "not_select", Description: "下記のキーワードリストを何も選択しない場合のみチェックを入れてください",
		})

		// カテゴリ内で何も登録されていない場合、「選択なし」にチェックを入れる
		var selectedFlag = false

		// 例外として「歌動画」が含まるカテゴリIDが指定された場合、「歌動画」キーワードもメニューに追加する
		if category.ID == SongVideoCategoryId {
			if slices.Contains(assignedRoleIDs, SongVideoRoleId) {
				menu = append(menu, tempest.SelectMenuOption{Label: "歌動画", Value: SongVideoRoleId, Default: true})
				selectedFlag = true
			} else {
				menu = append(menu, tempest.SelectMenuOption{Label: "歌動画", Value: SongVideoRoleId})
			}
		}

		for _, k := range keywordsMap[category.ID] {
			if slices.Contains(assignedRoleIDs, k.RoleID) {
				menu = append(menu, tempest.SelectMenuOption{Label: k.Name, Value: k.RoleID, Default: true})
				selectedFlag = true
			} else {
				menu = append(menu, tempest.SelectMenuOption{Label: k.Name, Value: k.RoleID})
			}
		}
		// カテゴリ内でキーワードを何も登録していない場合、「選択なし」にチェックが入っているようにする
		if !selectedFlag {
			menu[0].Default = true
		}

		layoutComponents = append(layoutComponents, tempest.LabelComponent{
			Type:  tempest.LABEL_COMPONENT_TYPE,
			Label: category.Name,
			Component: tempest.StringSelectComponent{
				Type:      tempest.STRING_SELECT_COMPONENT_TYPE,
				CustomID:  category.ID,
				Options:   menu[:],
				MinValues: 0,
				MaxValues: uint8(len(menu[:])),
			},
		})
	}

	data := tempest.ResponseModalData{
		CustomID:   "select_modal",
		Title:      "通知管理",
		Components: layoutComponents,
	}
	b, err := json.Marshal(data)
	log.Println(string(b))
	err = itx.AcknowledgeWithModal(data)

	if err != nil {
		log.Println("failed to acknowledge static component", err)
		return
	}
}

func selectedKeywordComponent(itx tempest.ModalInteraction) {
	guildID := itx.GuildID.String()
	targetUserID := itx.Member.User.ID.String()

	log.Println(guildID, targetUserID)

	// ロールとカテゴリの紐づき情報をDBから取得
	var keywords []internal.Keyword
	err := db.Select(&keywords, "SELECT name, role_id, category_id FROM keywords")
	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}
	// 扱いやすいようにMAPに変換　キー：カテゴリID、値：ロールID,名前
	roleMap := make(map[string]string)
	for _, k := range keywords {
		roleMap[k.RoleID] = k.Name
	}
	// 歌動画はkeywordsテーブルに含まれていないため、事前に追加しておく
	roleMap[SongVideoRoleId] = "歌動画"

	// カテゴリごとに選択されたロールIDリストを取得
	selectedRoleIDs := make(map[string][]string)
	for _, com := range itx.Data.Components {
		if label, ok := com.(tempest.LabelComponent); ok {
			if sl, ok := label.Component.(tempest.StringSelectComponent); ok {
				if len(sl.Values) != 0 && slices.Contains(sl.Values, "not_select") {
					selectedRoleIDs[sl.CustomID] = []string{}
				} else {
					selectedRoleIDs[sl.CustomID] = sl.Values
				}
			}
		}
	}

	// 後続の処理で使用　キーがロールID、値がカテゴリ
	cmap := make(map[string]string)
	for _, v := range keywords {
		cmap[v.RoleID] = v.CategoryID
	}
	// 歌動画
	cmap[SongVideoRoleId] = SongVideoCategoryId

	// ユーザに付与されているロールを取得
	// カテゴリごとにロールを分ける
	assignedRoleIDs := make(map[string][]string)
	for _, id := range itx.Member.RoleIDs {
		// カテゴリID
		cid := cmap[id.String()]
		assignedRoleIDs[cid] = append(assignedRoleIDs[cid], id.String())
	}

	log.Println("selectedRoleIDs", selectedRoleIDs)
	log.Println("assignedRoleIDs", assignedRoleIDs)

	var newSelectedRoleIDs []string
	var newNotSelectedRoleIDs []string

	for cid, _ := range selectedRoleIDs {
		// 選択したキーワードが登録済みキーワード一覧に存在しないものを取得
		newSelectedRoleIDs = append(newSelectedRoleIDs, slicesDiff(selectedRoleIDs[cid], assignedRoleIDs[cid])...)
		// 登録済みキーワードが選択したキーワード一覧に存在しないものを取得
		newNotSelectedRoleIDs = append(newNotSelectedRoleIDs, slicesDiff(assignedRoleIDs[cid], selectedRoleIDs[cid])...)
	}

	log.Println("newSelectedRoleIDs", newSelectedRoleIDs)
	log.Println("newNotSelectedRoleIDs", newNotSelectedRoleIDs)

	for _, rid := range newSelectedRoleIDs {
		slog.Info("GuildMemberRoleAdd",
			slog.String("role_id", rid),
			slog.String("role_name", roleMap[rid]),
			slog.String("guild_id", guildID),
			slog.String("user_id", targetUserID),
		)
		err := s.GuildMemberRoleAdd(guildID, targetUserID, rid)
		if err != nil {
			slog.Error(err.Error())
			panic(err)
		}
		log.Println("ロールを付与しました：", roleMap[rid])
	}

	for _, rid := range newNotSelectedRoleIDs {
		slog.Info("GuildMemberRoleRemove",
			slog.String("role_id", rid),
			slog.String("role_name", roleMap[rid]),
			slog.String("guild_id", guildID),
			slog.String("user_id", targetUserID),
		)
		err := s.GuildMemberRoleRemove(guildID, targetUserID, rid)
		if err != nil {
			slog.Error(err.Error())
			panic(err)
		}
		log.Println("ロールを取り消しました：", roleMap[rid])
	}

	// ユーザに登録・解除したキーワードを表示させる
	registerKeywords := ""
	for _, rid := range newSelectedRoleIDs {
		registerKeywords = registerKeywords + "- " + roleMap[rid] + "\n"
	}
	unRegisterKeywords := ""
	for _, rid := range newNotSelectedRoleIDs {
		unRegisterKeywords = unRegisterKeywords + "- " + roleMap[rid] + "\n"
	}

	content := "処理が完了しました。\n"
	if registerKeywords != "" {
		content = content + "### 通知登録したキーワード\n" + registerKeywords
	}
	if unRegisterKeywords != "" {
		content = content + "### 通知解除したキーワード\n" + unRegisterKeywords
	}

	err = itx.AcknowledgeWithMessage(tempest.ResponseMessageData{
		Content: content,
	}, true)
	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}
}

func slicesDiff(target, s []string) []string {
	var r []string
	for _, t := range target {
		if !slices.Contains(s, t) {
			r = append(r, t)
		}
	}
	return r
}

var songAddCommand = tempest.Command{
	Name:        "song-add",
	Description: "指定した動画を歌みたで告知する",
	Options: []tempest.CommandOption{
		{
			Name:        "url",
			Description: "動画のURL",
			Type:        tempest.STRING_OPTION_TYPE,
			Required:    true,
		},
	},
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		err := itx.Defer(true)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		url, _ := itx.GetOptionValue("url")
		err = AddSong(url.(string))
		if err != nil {
			slog.Error(err.Error())
			itx.SendLinearReply(errorMessage, true)
		}
		itx.SendLinearReply(successMessage, true)
	},
}

var keywordAddCommand = tempest.Command{
	Name:        "keyword-add",
	Description: "指定したキーワードを追加",
	Options: []tempest.CommandOption{
		{
			Name:        "category_id",
			Description: "カテゴリID",
			Type:        tempest.STRING_OPTION_TYPE,
			Required:    true,
		},
		{
			Name:        "keyword",
			Description: "キーワード",
			Type:        tempest.STRING_OPTION_TYPE,
			Required:    true,
		},
	},
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		err := itx.Defer(true)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		categoryID, _ := itx.GetOptionValue("category_id")
		keyword, _ := itx.GetOptionValue("category_id")
		err = AddKeyword(keyword.(string), categoryID.(string))
		if err != nil {
			slog.Error(err.Error())
			itx.SendLinearReply(errorMessage, true)
		}
		itx.SendLinearReply(successMessage, true)
	},
}

// AddSong
// 歌動画か判断できなかったものを目視で確認して、歌動画だったものを告知するようにタスクを登録するための処理
func AddSong(url string) error {
	// urlが https://www.youtube.com/watch?v=C56ImfpThK0 の形式であるため、=で分割して2つ目の要素を取得
	vid := strings.Split(url, "=")[1]

	yt, err := internal.NewYoutube(os.Getenv("YOUTUBE_API_KEY"))
	if err != nil {
		return err
	}

	videos, err := yt.Videos([]string{vid})
	if err != nil {
		return err
	}

	log.Println(videos)

	// HTTP経由でsong機能にアクセス
	songUrl := os.Getenv("SONG_TASK_URL") + "/song?v=" + vid
	resp, err := http.Post(songUrl, "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("失敗しました。status:" + resp.Status)
	}

	return nil
}

func AddKeyword(keyword string, categoryID string) error {
	guildID := os.Getenv("DISCORD_GUILD_ID")
	channel, err := s.GuildChannelCreate(guildID, keyword, discordgo.ChannelTypeGuildText)
	if err != nil {
		return err
	}

	_, err = s.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{ParentID: categoryID})
	if err != nil {
		return err
	}

	role, err := s.GuildRoleCreate(guildID, &discordgo.RoleParams{Name: keyword})
	if err != nil {
		return err
	}

	k := internal.Keyword{
		RoleID:        role.ID,
		Name:          keyword,
		CategoryID:    categoryID,
		ChannelID:     channel.ID,
		InclusionList: keyword,
	}

	_, err = db.NamedExec(internal.InsertKeywordsQuery, k)
	if err != nil {
		return err
	}

	return nil
}
