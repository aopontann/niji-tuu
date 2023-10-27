package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/dghubble/oauth1"
)

type Misskey struct {
	token string
}

type ReqBody struct {
	I      string `json:"i"`
	Text   string `json:"text"`
	Detail bool   `json:"detail"`
}

type Twitter struct {
	vid   string
	title string
}

func NewTwitter() *Twitter {
	return &Twitter{}
}

func (tw *Twitter) Id(vid string) *Twitter {
	tw.vid = vid
	return tw
}

func (tw *Twitter) Title(title string) *Twitter {
	tw.title = title
	return tw
}

func (tw *Twitter) Tweet() error {
	url := "https://api.twitter.com/2/tweets"
	config := oauth1.NewConfig(os.Getenv("TWITTER_API_KEY"), os.Getenv("TWITTER_API_SECRET_KEY"))
	token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"))

	reqBody := fmt.Sprintf(`{"text": "【5分後に公開】\n\n%s\n\nhttps://www.youtube.com/watch?v=%s"}`, tw.title, tw.vid)
	payload := strings.NewReader(reqBody)

	httpClient := config.Client(oauth1.NoContext, token)

	resp, err := httpClient.Post(url, "application/json", payload)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Raw Response Body:\n%v\n", string(body))
	return nil
}

func NewMisskey(token string) *Misskey {
	return &Misskey{token: token}
}

func (m *Misskey) Post(id string, title string) error {
	url := "https://@aopontan@misskey.io/api/notes/create"
	content := fmt.Sprintf(`
	【5分後に公開】
	%s
	https://www.youtube.com/watch?v=%s
	`, title, id)

	resb := ReqBody{
		I:      m.token,
		Text:   content,
		Detail: false,
	}

	payload, err := json.Marshal(resb)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}