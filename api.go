package ffbinaries

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const APIURL = "https://ffbinaries.com/api/v1/version/"

const (
	windows = "windows-64"
	linux   = "linux-64"
	darwin  = "osx-64"
)

type Binaries struct {
	FFmpeg  string `json:"ffmpeg,omitempty"`
	FFprobe string `json:"ffprobe,omitempty"`
	FFplay  string `json:"ffplay,omitempty"`
}

type Response struct {
	Version   string              `json:"version"`
	Permalink string              `json:"permalink"`
	Bin       map[string]Binaries `json:"bin"`
}

func unmarshal(body io.ReadCloser, b any) error {
	buf, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(buf, b); err != nil {
		return err
	}
	return nil
}

func osKey() string {
	switch runtime.GOOS {
	case "windows":
		return windows
	case "darwin":
		return darwin
	case "linux":
		return linux
	default:
		return ""
	}
}

func getDownloadUrl(product string, version string) (string, error) {
	// Check product
	if product != "ffmpeg" && product != "ffprobe" {
		return "", fmt.Errorf("invalid product, must be ffmpeg or ffprobe")
	}

	if version == "" {
		version = "latest"
	}

	// Build de URL
	fullurl := APIURL + version

	// Query the API
	resp, err := http.Get(fullurl)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	body := &Response{}

	// Parse Body
	if err := unmarshal(resp.Body, body); err != nil {
		return "", err
	}

	os := osKey()
	if os == "" {
		return "", fmt.Errorf("os not found")
	}

	bin, ok := body.Bin[os]
	if !ok {
		return "", fmt.Errorf("bin not found for OS %s", os)
	}

	download := ""
	switch product {
	case "ffmpeg":
		download = bin.FFmpeg
	case "ffplay":
		download = bin.FFplay
	case "ffprobe":
		download = bin.FFprobe
	}

	if download != "" {
		return download, nil
	}
	return "", fmt.Errorf("unable to found %s for %s", product, os)
}

func downloadZip(url string, product string, dstPath string) (string, error) {

	if dstPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dstPath = wd
	}
	dstPath = filepath.Join(dstPath, fmt.Sprintf("%s.zip", product))

	file, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	return dstPath, nil
}

func unzipProduct(file string, dstPath string) (string, error) {
	archive, err := zip.OpenReader(file)
	if err != nil {
		return "", err
	}
	dst := ""
	for _, f := range archive.File {
		dst = filepath.Join(dstPath, f.Name)
		file, err := os.Create(dst)
		if err != nil {
			return "", err
		}
		defer file.Close()
		src, err := f.Open()
		if err != nil {
			return "", err
		}
		defer src.Close()
		if _, err := io.Copy(file, src); err != nil {
			return "", err
		}
	}
	defer os.Chmod(dst, 0700)
	return dst, nil
}

func Download(product string, version string, dstPath string) (string, error) {
	url, err := getDownloadUrl(product, version)
	if err != nil {
		return "", nil
	}
	zip, err := downloadZip(url, product, dstPath)
	if err != nil {
		return "", nil
	}
	defer os.Remove(zip)
	return unzipProduct(zip, dstPath)
}
