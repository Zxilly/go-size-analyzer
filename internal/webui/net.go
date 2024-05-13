//go:build !embed

package webui

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	gsa "github.com/Zxilly/go-size-analyzer"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

const BaseURL = "https://github.com/Zxilly/go-size-analyzer/releases/download/ui-v" +
	gsa.StaticVersion +
	"/index.html"

func GetTemplate() string {
	tmpl, err := readTemplate()
	if err != nil {
		utils.FatalError(err)
	}
	return tmpl
}

func (UpdateCacheFlag) BeforeReset(app *kong.Kong, _ kong.Vars) error {
	p, err := getCacheFilePath()
	if err != nil {
		return err
	}

	_, err = updateCache(p)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(app.Stdout, "Cache updated: %s\n", p)
	if err != nil {
		return err
	}
	app.Exit(0)
	return nil
}

func download() (string, error) {
	slog.Info("Downloading template")
	resp, err := http.Get(BaseURL)
	if err != nil {
		return "", err
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(resp.Body)

	// check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download template: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func getCacheFilePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(dir, "go-size-analyzer")

	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return "", err
	}

	file := filepath.Join(cacheDir, fmt.Sprintf("webui-v%s.html", gsa.StaticVersion))

	return file, nil
}

func readTemplate() (string, error) {
	cacheFile, err := getCacheFilePath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		tmpl, err := updateCache(cacheFile)
		if err != nil {
			return "", err
		}
		return tmpl, nil
	}

	tmplBytes, err := os.ReadFile(cacheFile)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to read cache file: %v", err))

		// still try to download the template
		tmpl, err := download()
		if err != nil {
			return "", err
		}
		return tmpl, nil
	}
	return string(tmplBytes), nil
}

func updateCache(cacheFilePath string) (string, error) {
	tmpl, err := download()
	if err != nil {
		return "", err
	}

	err = os.WriteFile(cacheFilePath, []byte(tmpl), 0600)
	if err != nil {
		return "", err
	}

	return tmpl, nil
}
