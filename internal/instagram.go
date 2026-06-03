package internal

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
)

const INSTAGRAM_URL = "https://graph.instagram.com/v22.0"

type Instagram struct {
	ID    string
	Token string
}

type InstagramResponse struct {
	Id string `json:"id"`
}

type InstagramStatusResponse struct {
	Id         string `json:"id"`
	Status     string `json:"status"`
	StatusCode string `json:"status_code"`
}

func NewInstagram(id string, token string) *Instagram {
	return &Instagram{
		ID:    id,
		Token: token,
	}
}

func (i *Instagram) Notification(v *NotificationVideo) error {
	// 投稿
	idContainerID, err := i.post(v)
	if err != nil {
		return err
	}

	// 画像ファイルのアップロードが完了するまで定期的に確認
	err = retry.Do(
		func() error {
			finished, err := i.isFinished(idContainerID)
			if err != nil {
				return err
			}
			if !finished {
				return errors.New("not finished")
			}
			return nil
		},
		retry.Attempts(3),
		retry.Delay(5*time.Second),
	)
	if err != nil {
		return err
	}

	// 投稿を公開
	err = i.publish(idContainerID)
	if err != nil {
		return err
	}

	return nil
}

func (i *Instagram) post(v *NotificationVideo) (string, error) {
	req, err := http.NewRequest(http.MethodPost, INSTAGRAM_URL+"/"+i.ID+"/media", nil)
	if err != nil {
		return "", err
	}
	// リクエスト内容を付与
	q := req.URL.Query()
	q.Add("access_token", i.Token)
	q.Add("image_url", v.Thumbnail)
	q.Add("caption", "5分後に公開\n"+v.Title)

	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"instagram api error",
			slog.Int("code", resp.StatusCode),
			slog.String("msg", string(body)),
		)
		return "", errors.New("instagram api post error")
	}

	var resMediaBody InstagramResponse
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &resMediaBody)
	if err != nil {
		return "", err
	}

	return resMediaBody.Id, nil
}

func (i *Instagram) isFinished(id string) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, INSTAGRAM_URL+"/"+id+"?fields=status,status_code", nil)
	if err != nil {
		return false, err
	}
	q := req.URL.Query()
	q.Add("access_token", i.Token)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"instagram api error",
			slog.Int("code", resp.StatusCode),
			slog.String("msg", string(body)),
		)
		return false, errors.New("instagram api status error")
	}

	var resMediaStatusBody InstagramStatusResponse
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &resMediaStatusBody)
	if err != nil {
		return false, err
	}

	return resMediaStatusBody.Status == "FINISHED", nil
}

func (i *Instagram) publish(id string) error {
	req, err := http.NewRequest(http.MethodPost, INSTAGRAM_URL+"/"+i.ID+"/media_publish", nil)
	if err != nil {
		return err
	}
	// リクエスト内容を付与
	q := req.URL.Query()
	q.Add("access_token", i.Token)
	q.Add("creation_id", id)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"instagram api error",
			slog.Int("code", resp.StatusCode),
			slog.String("msg", string(body)),
		)
		return errors.New("instagram api publish error")
	}

	return nil
}
