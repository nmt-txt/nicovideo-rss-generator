package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// filters追加など拡張性確保のため
type SearchQuery struct {
	Query string `json:"query"`
}

type System struct {
	Version string
}

type RssGenerator struct {
	Title                       string `json:"title"`
	Description                 string `json:"description"`
	Link                        string `json:"link"`
	ShouldSuppressSystemMessage bool   `json:"shouldSuppressSystemMessage"`
}

type VideoFetcher struct {
	ShouldFetchThumbnail bool `json:"shouldFetchThumbnail"`
}

type Config struct {
	SearchQueries []SearchQuery `json:"searchQueries"`
	Log           string        `json:"log,omitempty"`
	System        System        `json:"-"`
	RssGenerator  RssGenerator  `json:"rssGenerator"`
	VideoFetcher  VideoFetcher  `json:"videoFetcher"`
}

// LoadConfig 設定ファイルを読み込み、検証して返す。
func LoadConfig(fileContents []byte) (*Config, error) {
	var cfg Config
	// デフォルト設定
	cfg.Log = "info"
	cfg.System = System{Version: "1.0.1"}
	cfg.RssGenerator = RssGenerator{
		Title:                       "Nicovideo RSS DIY",
		Description:                 "ニコニコ動画新着RSS(自作)",
		Link:                        "https://www.nicovideo.jp/",
		ShouldSuppressSystemMessage: false,
	}
	cfg.VideoFetcher = VideoFetcher{
		ShouldFetchThumbnail: false,
	}

	// アンマーシャル
	if err := json.Unmarshal(fileContents, &cfg); err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// 調整
	cfg.Log = strings.ToLower(strings.TrimSpace(cfg.Log))
	for i := range cfg.SearchQueries {
		cfg.SearchQueries[i].Query = strings.TrimSpace(cfg.SearchQueries[i].Query)
	}

	if err := cfg.verify(); err != nil {
		return nil, fmt.Errorf("設定ファイルの内容が不正です: %w", err)
	}

	return &cfg, nil
}

func (c Config) verify() error {

	switch c.Log {
	case "debug", "info", "error":
	default:
		return fmt.Errorf("logはdebug/info/errorのいずれかである必要があります。")
	}

	for _, q := range c.SearchQueries {
		if q.Query == "" {
			return fmt.Errorf("検索タグ内容を空にすることはできません。APIガイドを参照してください(https://site.nicovideo.jp/search-api-docs/snapshot)。(任意のfilters併用は未対応です)")
		}
	}
	return nil
}
