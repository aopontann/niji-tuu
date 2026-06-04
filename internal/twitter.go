package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/dghubble/oauth1"
)

type Twitter struct {
	baseURL string
	config  *oauth1.Config
	token   *oauth1.Token
}

type TwitterImageUploadRequest struct {
	Media         string `json:"media"`          // Base64エンコードされた文字列
	MediaCategory string `json:"media_category"` // Base64エンコードされた文字列
}

type TwitterMedia struct {
	MediaIDs []string `json:"media_ids"`
}

type TwitterPostRequest struct {
	Text  string       `json:"text"`
	Media TwitterMedia `json:"media"`
}

type TwitterUploadResponse struct {
	Data struct {
		ExpiresAfterSecs int    `json:"expires_after_secs"`
		ID               string `json:"id"`
		MediaKey         string `json:"media_key"`
		ProcessingInfo   struct {
			CheckAfterSecs  int `json:"check_after_secs"`
			ProgressPercent int `json:"progress_percent"`
		} `json:"processing_info"`
		Size int `json:"size"`
	} `json:"data"`
	Errors []struct {
		Title  string `json:"title"`
		Type   string `json:"type"`
		Detail string `json:"detail"`
		Status int    `json:"status"`
	} `json:"errors"`
}

func NewTwitter(consumerKey, consumerSecret, accessToken, accessSecret string) *Twitter {
	return &Twitter{
		baseURL: "https://api.x.com/2",
		config:  oauth1.NewConfig(consumerKey, consumerSecret),
		token:   oauth1.NewToken(accessToken, accessSecret),
	}
}

func (t *Twitter) Notification(v *NotificationVideo) error {
	mediaID, err := t.upload(v.Thumbnail)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	slog.Info(mediaID)
	err = t.post(v.Title, mediaID)
	return nil
}

// サムネイル画像をTwitterのサーバーにアップロードし、専用のIDを取得
func (t *Twitter) upload(url string) (string, error) {
	// サムネイルをダウンロード
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	imageDataBase64 := base64.StdEncoding.EncodeToString(imageData)
	reqBody := TwitterImageUploadRequest{
		Media:         imageDataBase64,
		MediaCategory: "tweet_image",
	}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Twitterのサーバーにアップロード
	uploadURL := t.baseURL + "/media/upload"
	client := t.config.Client(oauth1.NoContext, t.token)
	req, err := http.NewRequest("POST", uploadURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// リクエスト実行
	postResp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer postResp.Body.Close()

	b, err := io.ReadAll(postResp.Body)
	if err != nil {
		return "", err
	}

	if postResp.StatusCode != http.StatusOK {
		return "", errors.New(string(b))
	}

	var data TwitterUploadResponse
	err = json.Unmarshal(b, &data)
	if err != nil {
		return "", err
	}

	return data.Data.ID, nil
}

func (t *Twitter) post(title, mediaID string) error {
	reqBody := TwitterPostRequest{
		Text:  title,
		Media: TwitterMedia{MediaIDs: []string{mediaID}},
	}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	tweetsURL := t.baseURL + "/tweets"
	client := t.config.Client(oauth1.NoContext, t.token)
	req, err := http.NewRequest("POST", tweetsURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// リクエスト実行
	postResp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	b, err := io.ReadAll(postResp.Body)
	if err != nil {
		return err
	}

	if postResp.StatusCode != http.StatusCreated {
		return errors.New(string(b))
	}

	return nil
}
