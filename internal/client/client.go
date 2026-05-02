package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"nicovideoRSSDIY/internal/repository"
	"strings"
	"time"
)

// todo: HTTPエラー周り処理の重複が多い。共通化

type VideoClient struct {
	httpClient *http.Client
	baseURL    string
	UserAgent  string
}

var (
	ErrFiltersFormat    = errors.New("filtersパラメータの書式が間違っています")
	ErrRespQueryParse   = errors.New("リクエストに不正なパラメーターがあります")
	ErrRespInternal     = errors.New("サーバーの異常です。")
	ErrRespMaintainance = errors.New("サービスがメンテナンス中です。メンテナンス終了までお待ち下さい。")
	ErrNotFound         = errors.New("見つかりませんでした。削除済みの可能性があります。")
)

func NewVideoClient(baseURL string, userAgent string) *VideoClient {
	return &VideoClient{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		UserAgent:  userAgent,
	}
}

type ResponseMetadata struct {
	Status       int    `json:"status"`
	ID           string `json:"id,omitempty"`
	TotalCount   int    `json:"totalCount,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}
type SearchVideoResponse struct {
	Meta   ResponseMetadata    `json:"meta"`
	Videos []*repository.Video `json:"data,omitempty"`
}

// SearchVideo 動画検索APIを呼び出す。tagExact検索である。filtersは"[フィールド名][演算子]=値"の形式で指定する。その他必要なものは関数内でセットされる。
func (c *VideoClient) SearchVideo(ctx context.Context, query string, filters []string) (*SearchVideoResponse, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("_limit", "100")
	params.Set("_sort", "-startTime")
	params.Set("targets", "tagsExact")
	params.Set("fields", "contentId,title,description,thumbnailUrl,startTime,tags")
	params.Set("_context", c.UserAgent)

	for _, filter := range filters {
		splited := strings.Split(filter, "=")
		if len(splited) != 2 {
			return nil, fmt.Errorf("filtersパラメーターをセットできません: %w", ErrFiltersFormat)
		}
		params.Set(fmt.Sprintf("filters%s", splited[0]), splited[1])
	}

	urlStr, err := url.JoinPath(c.baseURL, "video", "contents", "search")
	if err != nil {
		return nil, fmt.Errorf("URLを作成できません: %w", err)
	}

	// log.Print(urlStr)
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("コンテキストを作成できません: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 注意: Do()は2xx以外でもエラーを返さない。下記は必要
	switch resp.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("動画情報取得に失敗しました: %w", ErrRespInternal)
	case http.StatusServiceUnavailable:
		return nil, fmt.Errorf("動画情報取得に失敗しました: %w", ErrRespMaintainance)
	default:
		return nil, fmt.Errorf("動画情報取得に不明なエラーで失敗しました: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	var body io.Reader = resp.Body

	// body = io.TeeReader(resp.Body, os.Stderr) // for debugging

	var respoData SearchVideoResponse
	if err := json.NewDecoder(body).Decode(&respoData); err != nil {
		return nil, fmt.Errorf("レスポンスのデコードに失敗しました: %w", err)
	}

	switch respoData.Meta.Status {
	case 200:
		return &respoData, nil
	case 400:
		return nil, fmt.Errorf("エラーが返却されました: %w (HTTP status %d, %s)", ErrRespQueryParse, respoData.Meta.Status, respoData.Meta.ErrorMessage)
	case 500:
		return nil, fmt.Errorf("エラーが返却されました: %w (HTTP status %d, %s)", ErrRespInternal, respoData.Meta.Status, respoData.Meta.ErrorMessage)
	case 503:
		return nil, fmt.Errorf("エラーが返却されました: %w (HTTP status %d, %s)", ErrRespMaintainance, respoData.Meta.Status, respoData.Meta.ErrorMessage)
	default:
		return nil, fmt.Errorf("不明なエラーが発生しました: HTTP status code %d, %s", respoData.Meta.Status, respoData.Meta.ErrorMessage)
	}
}

type LastModified struct {
	LastModified time.Time `json:"last_modified"`
}

func (c *VideoClient) FetchLastModified(ctx context.Context) (time.Time, error) {
	urlStr, err := url.JoinPath(c.baseURL, "version")
	if err != nil {
		return time.Now(), fmt.Errorf("URLを作成できません: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return time.Now(), fmt.Errorf("コンテキストを作成できません: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return time.Now(), err
	}
	defer resp.Body.Close()

	// 注意: Do()は2xx以外でもエラーを返さない。下記は必要
	switch resp.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusInternalServerError:
		return time.Now(), fmt.Errorf("データ切り替え日時情報取得に失敗しました: %w", ErrRespInternal)
	case http.StatusServiceUnavailable:
		return time.Now(), fmt.Errorf("データ切り替え日時情報取得に失敗しました: %w", ErrRespMaintainance)
	default:
		return time.Now(), fmt.Errorf("データ切り替え日時情報取得に不明なエラーで失敗しました: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	var body io.Reader = resp.Body

	var respData LastModified
	if err := json.NewDecoder(body).Decode(&respData); err != nil {
		return time.Now(), fmt.Errorf("レスポンスのデコードに失敗しました: %w", err)
	}

	return respData.LastModified, nil
}

type thumbnailMeta struct {
	URL    string
	Type   string
	Length int64
}

type ThumbnailClient struct {
	httpClient *http.Client
	UserAgent  string
}

func NewThumbnailClient(userAgent string) *ThumbnailClient {
	return &ThumbnailClient{
		httpClient: &http.Client{},
		UserAgent:  userAgent,
	}
}

// FetchThumbnailMeta サムネイルのTypeとLengthを取得する。Lengthはサーバーが提供しない場合-1になる可能性がある
func (c *ThumbnailClient) FetchThumbnailMeta(ctx context.Context, thumbnailURL string) (*thumbnailMeta, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", thumbnailURL, nil)
	if err != nil {
		return nil, fmt.Errorf("サムネイルのHEADリクエストを作成できません: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("サムネイル情報取得のHEADリクエストに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	// 注意: Do()は2xx以外でもエラーを返さない。下記は必要
	switch resp.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("サムネイル情報取得に失敗しました: %w", ErrRespInternal)
	case http.StatusServiceUnavailable:
		return nil, fmt.Errorf("サムネイル情報取得に失敗しました: %w", ErrRespMaintainance)
	case http.StatusNotFound:
		return nil, fmt.Errorf("サムネイル情報取得に失敗しました: %w", ErrNotFound)
	default:
		return nil, fmt.Errorf("サムネイル情報取得に不明なエラーで失敗しました: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	meta := &thumbnailMeta{
		URL:    resp.Request.URL.String(),
		Type:   resp.Header.Get("Content-Type"),
		Length: resp.ContentLength,
	}

	return meta, nil
}
