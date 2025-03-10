package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	githubRepo    = "mpcastillo2023/box-office-investigation"
	cacheDuration = 5 * time.Minute
)

type Release struct {
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	PubDate   string `json:"pub_date"`
	Url       string `json:"url"`
	Signature string `json:"signature"`
}

var (
	releaseCache *Release
	// cacheMutex   sync.Mutex
	// cacheExpires time.Time
)

func getLatestGHRelease(platform string) (*Release, error) {
	// cacheMutex.Lock()
	// defer cacheMutex.Unlock()

	// if time.Now().Before(cacheExpires) {
	// 	return releaseCache, nil
	// }

	url := "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releaseData map[string]interface{}
	if err := json.Unmarshal(body, &releaseData); err != nil {
		return nil, err
	}

	version := releaseData["tag_name"].(string)
	notes := releaseData["body"].(string)
	notes = strings.TrimSuffix(notes, "See the assets to download this version and install.")
	notes = strings.TrimSpace(notes)

	pubDate := releaseData["published_at"].(string)

	platformsExtensions := map[string]string{
		"linux-x86_64":   "amd64.AppImage.gz",
		"darwin-x86_64":  "app.tar.gz",
		"darwin-aarch64": "app.tar.gz",
		"windows-x86_64": "x64_en-US.msi",
	}
	extension := platformsExtensions[platform]

	var updateDownloadUrl string
	var updateSignature string
	if assets, ok := releaseData["assets"].([]interface{}); ok {
		for _, asset := range assets {
			assetMap := asset.(map[string]interface{})
			assetName := assetMap["name"].(string)
			assetURL := assetMap["browser_download_url"].(string)
			if strings.HasSuffix(assetName, extension) {
				updateDownloadUrl = assetURL
			} else if strings.HasSuffix(assetName, extension+".sig") {
				resp, err := http.Get(assetURL)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}

				updateSignature = string(body)
			}
		}
	}

	releaseCache = &Release{
		Version:   version,
		Notes:     notes,
		PubDate:   pubDate,
		Url:       updateDownloadUrl,
		Signature: updateSignature,
	}
	// cacheExpires = time.Now().Add(cacheDuration)

	return releaseCache, nil
}

func getUpdaterHandler(c *gin.Context) {
	platform := c.Param("platform")
	currentVersion := c.Param("current_version")

	release, err := getLatestGHRelease(platform)
	if err != nil || release == nil {
		c.Status(http.StatusNoContent)
		return
	}

	if release.Version == "v"+currentVersion {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, release)
}

func main() {
	r := gin.Default()
	r.GET("/tauri-releases/box-office-investigation/:platform/:current_version", getUpdaterHandler)

	r.Run(":8080")
}
