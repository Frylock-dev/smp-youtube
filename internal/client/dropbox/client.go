package dropbox

import (
	"bytes"
	"fmt"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/sharing"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"smp-youtube/internal/client/dropbox/model"
	"strings"
)

const (
	chunkSize = 8 * 1024 * 1024
)

type Client struct {
	refreshToken string
	appKey       string
	appSecret    string
}

func NewClient(
	refreshToken string,
	appKey string,
	appSecret string,
) *Client {
	return &Client{
		refreshToken: refreshToken,
		appKey:       appKey,
		appSecret:    appSecret,
	}
}
func (c *Client) preparePath(path string) string {
	path = strings.TrimSpace(path)
	path = filepath.ToSlash(filepath.Clean(path))
	path = strings.ReplaceAll(path, "../", "")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range invalidChars {
		path = strings.ReplaceAll(path, char, "_")
	}
	return path
}

func (c *Client) Upload(path string, remotePath string) error {
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return err
	}

	remotePath = c.preparePath(remotePath)

	config := dropbox.Config{
		Token: accessToken,
	}

	client := files.New(config)

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Error(err)
		}
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("ошибка получения информации о файле: %v", err)
	}

	sessStart := files.NewUploadSessionStartArg()
	sessStart.Close = false

	sessionStartResult, err := client.UploadSessionStart(sessStart, nil)
	if err != nil {
		return fmt.Errorf("ошибка запуска сессии загрузки: %v", err)
	}

	uploadSessionCursor := files.NewUploadSessionCursor(sessionStartResult.SessionId, 0)

	buffer := make([]byte, chunkSize)

	for totalBytesRead := int64(0); totalBytesRead < fileInfo.Size(); {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("ошибка чтения файла: %v", err)
		}

		if bytesRead == 0 {
			break
		}

		sessAppend := files.NewUploadSessionAppendArg(uploadSessionCursor)
		sessAppend.Close = totalBytesRead+int64(bytesRead) >= fileInfo.Size()

		err = client.UploadSessionAppendV2(sessAppend, bytes.NewReader(buffer[:bytesRead]))
		if err != nil {
			return fmt.Errorf("ошибка добавления части файла: %v", err)
		}

		uploadSessionCursor.Offset += uint64(bytesRead)

		totalBytesRead += int64(bytesRead)
	}

	commit := files.NewCommitInfo(remotePath)
	_, err = client.UploadSessionFinish(
		&files.UploadSessionFinishArg{
			Cursor: uploadSessionCursor,
			Commit: commit,
		}, nil)

	if err != nil {
		return fmt.Errorf("ошибка завершения загрузки: %v", err)
	}

	return nil
}

func (c *Client) GetAccessToken() (string, error) {
	var accessToken model.AccessToken

	client := resty.New()

	resp, err := client.
		R().
		SetFormData(map[string]string{
			"refresh_token": c.refreshToken,
			"grant_type":    "refresh_token",
			"client_id":     c.appKey,
			"client_secret": c.appSecret,
		}).
		SetResult(&accessToken).
		Post("https://api.dropbox.com/oauth2/token")
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf(
			"getAccessToken error status: %d body: %v",
			resp.StatusCode(),
			resp.Body(),
		)
	}

	return accessToken.Token, nil
}

func (c *Client) GetSharingLink(folderPath string) (string, error) {
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return "", err
	}

	config := dropbox.Config{
		Token: accessToken,
	}

	client := sharing.New(config)

	args := sharing.NewCreateSharedLinkWithSettingsArg(c.preparePath(folderPath))
	res, err := client.CreateSharedLinkWithSettings(args)
	if err != nil {
		log.Error(err)
	}

	result, ok := res.(*sharing.FolderLinkMetadata)
	if ok {
		return result.Url, nil
	}

	return "", nil
}
