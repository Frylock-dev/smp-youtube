package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"path/filepath"
)

func DownloadVideoWithCookies(url string, cookies string, output string, title string) error {
	outputTemplate := filepath.Join(output, fmt.Sprintf("%s.%%(ext)s", title))

	cmd := exec.Command(
		"yt-dlp",
		"--cookies", cookies,
		"-f", "bestvideo+bestaudio",
		"-o", outputTemplate,
		url,
	)

	result, err := cmd.CombinedOutput()
	if err != nil {
		log.Error(string(result))
	}

	log.WithFields(log.Fields{
		"url":   url,
		output:  output,
		"title": title,
	}).Println(string(result))

	return nil
}
