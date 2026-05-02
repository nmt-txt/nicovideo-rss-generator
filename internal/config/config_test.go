package config

import (
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	contents := []byte(`{
		"searchQueries": [
	        {"query": "  VOCALOID  "},
	        {"query": "ソフトウェアトーク劇場"}
	    ],
	    "log": "DEBUG"
	}`)
	cfg, err := LoadConfig(contents)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg.Log != "debug" {
		t.Fatalf("expected log debug, got %q", cfg.Log)
	}

	if cfg.SearchQueries[0].Query != "VOCALOID" {
		t.Fatalf("expected trimmed query VOCALOID, got %q", cfg.SearchQueries[0].Query)
	}
}

func TestLoadConfig_InvalidLog(t *testing.T) {
	contents := []byte(`{
	    "searchQueries": [{"query": "foo"}],
	    "log": "trace"
	}`)
	if _, err := LoadConfig(contents); err == nil {
		t.Fatalf("expected error for invalid log")
	}
}

func TestLoadConfig_InvalidQuery(t *testing.T) {
	contents := []byte(`{
	    "searchQueries": [{"query": ""}],
	    "log": "info"
	}`)
	if _, err := LoadConfig(contents); err == nil {
		t.Fatalf("expected error for invalid query")
	}
}

func TestLoadConfig_RSSGenerator(t *testing.T) {
	contents := []byte(`{
	    "searchQueries": [{"query": "foo"}],
	    "rssGenerator": {
	        "title": "Custom RSS Title",
	        "description": "Custom RSS Description",
	        "link": "https://example.com/custom"
	    }
	}`)
	cfg, err := LoadConfig(contents)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg.RssGenerator.Title != "Custom RSS Title" {
		t.Fatalf("expected \"Custom RSS Title\", got %q", cfg.RssGenerator.Title)
	}
	if cfg.RssGenerator.Description != "Custom RSS Description" {
		t.Fatalf("expected \"Custom RSS Description\", got %q", cfg.RssGenerator.Description)
	}
	if cfg.RssGenerator.Link != "https://example.com/custom" {
		t.Fatalf("expected \"https://example.com/custom\", got %q", cfg.RssGenerator.Link)
	}
}

func TestLoadConfig_VideoFetcher(t *testing.T) {
	contents := []byte(`{
	    "searchQueries": [{"query": "foo"}],
	    "videoFetcher": {
	        "shouldFetchThumbnail": true
	    }
	}`)
	cfg, err := LoadConfig(contents)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if !cfg.VideoFetcher.ShouldFetchThumbnail {
		t.Fatalf("expected shouldFetchThumbnail to be true")
	}
}

func TestLoadConfig_DefaultFallBack(t *testing.T) {
	contents := []byte(`{
	    "searchQueries": [{"query": "foo"}]
	}`)
	cfg, err := LoadConfig(contents)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg.Log != "info" {
		t.Fatalf("expected log default to info, got %q", cfg.Log)
	}

	// RSS Generator デフォルト確認
	if cfg.RssGenerator.Title != "Nicovideo RSS DIY" {
		t.Fatalf("expected default RSS title, got %q", cfg.RssGenerator.Title)
	}
	if cfg.RssGenerator.Description != "ニコニコ動画新着RSS(自作)" {
		t.Fatalf("expected default RSS description, got %q", cfg.RssGenerator.Description)
	}
	if cfg.RssGenerator.Link != "https://www.nicovideo.jp/" {
		t.Fatalf("expected default RSS link, got %q", cfg.RssGenerator.Link)
	}

	// Video Fetcher デフォルト確認
	if cfg.VideoFetcher.ShouldFetchThumbnail {
		t.Fatalf("expected default shouldFetchThumbnail to be false")
	}

	// System デフォルト確認(Ver不要, それ以外)
	if cfg.System.SnapShotAPIURL != "https://snapshot.search.nicovideo.jp/api/v2/snapshot" {
		t.Fatalf("expected default SnapShotAPIURL, got %q", cfg.System.SnapShotAPIURL)
	}
}
