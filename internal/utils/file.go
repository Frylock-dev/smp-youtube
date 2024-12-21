package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func FindFileByName(directory, name string) (string, error) {
	var foundFile string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasPrefix(info.Name(), name) {
			foundFile = path
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if foundFile == "" {
		return "", fmt.Errorf("файл с названием '%s' не найден", name)
	}

	return foundFile, nil
}

func SanitizeFilename(fileName string) string {
	reg := regexp.MustCompile(`[^\p{L}\p{N}\s.-]`)
	sanitizedBase := reg.ReplaceAllString(fileName, "")
	sanitizedBase = strings.ReplaceAll(strings.TrimSpace(sanitizedBase), " ", "_")
	reg = regexp.MustCompile(`\.+`)
	sanitizedBase = reg.ReplaceAllString(sanitizedBase, "_")
	return sanitizedBase
}
