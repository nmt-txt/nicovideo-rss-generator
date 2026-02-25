package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"nicovideoRSSDIY/internal/client"
	"nicovideoRSSDIY/internal/config"
	"nicovideoRSSDIY/internal/repository"
	"nicovideoRSSDIY/internal/rss"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configDirPath := filepath.Join("./")
	if len(os.Args) > 2 {
		panic("使い方: niconico-rss-diy [config_dir_path]")
	} else if len(os.Args) == 2 {
		configDirPath = os.Args[1]
	}

	configBytes, err := os.ReadFile(filepath.Join(configDirPath, "config.json"))
	if err != nil {
		panic(fmt.Sprintf("設定ファイルの読み込みに失敗しました: %v", err))
	}

	cfg, err := config.LoadConfig(configBytes)
	if err != nil {
		panic(fmt.Sprintf("設定ファイルの読込・解析に失敗しました: %v", err))
	}

	level := func(levelStr string) slog.Level {
		switch levelStr {
		case "debug":
			return slog.LevelDebug
		case "error":
			return slog.LevelError
		default:
			return slog.LevelInfo
		}
	}(cfg.Log)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	slog.Info(fmt.Sprintf("Nicovideo RSS DIY v%s", cfg.System.Version))
	slog.Info("ログレベル: " + strings.ToUpper(cfg.Log))

	slog.Info(fmt.Sprintf("検索クエリ: %d件", len(cfg.SearchQueries)))

	vRepo := repository.NewVideoRepository(200)
	nRepo := repository.NewNotificationRepository()
	rRepo := repository.NewRSSRepository()

	// 起動中表示
	nRepo.AddNotification(
		repository.NotificationInfo,
		"起動中...",
		errors.New("データを集めています。しばらくお待ちください。(クエリ数 + 3 分程度)"),
		false,
	)
	vRepo.AddSortedVideos([]*repository.Video{
		{
			ID:               "sm9",
			Title:            "新・豪血寺一族 -煩悩解放 - レッツゴー！陰陽師",
			Description:      "レッツゴー！陰陽師（フルコーラスバージョン）",
			StartTime:        time.Date(2007, 3, 6, 0, 33, 0, 0, time.FixedZone("JST", 9*60*60)),
			ThumbnailURL:     "https://nicovideo.cdn.nimg.jp/thumbnails/9/9",
			ThumbnailType:    "image/jpeg",
			ThumbnailLength:  6337,
			TagsConnectedStr: "陰陽師 レッツゴー！陰陽師 公式 音楽 ゲーム 弾幕動画 伝説 最古の動画 3月6日投稿動画 重要ニコニコ文化財 sm9",
		},
	})

	rssBytes, err := rss.GenerateRSS(nRepo.Notifications, vRepo.Videos, &cfg.RssGenerator)
	if err != nil {
		panic(fmt.Sprintf("RSSの生成に失敗しました: %v", err))
	}
	rRepo.SetFeed(rssBytes)

	// HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// これでいいのか?
		slog.Info("HTTP_REQUEST", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.String("user-agent", r.UserAgent()))
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		w.Header().Set("ETag", rRepo.Etag())
		http.ServeContent(w, r, "feed.xml", rRepo.ModifiedAt(), rRepo.Feed())
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: nil,
	}
	slog.Info("listening on 8080")
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			panic(fmt.Sprintf("HTTPサーバーの起動に失敗しました: %v", err))
		}
	}()

	go worker(ctx, vRepo, nRepo, rRepo, cfg)

	// シャットダウン
	<-ctx.Done()
	slog.Debug("signal received")
	stop()
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 20*time.Second)
	defer shutdownRelease()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error(fmt.Sprintf("HTTPサーバーのシャットダウンに失敗しました: %v", err))
	}
	slog.Info("exiting")
}

// worker 動画・サムネイル情報収集及びRSS生成貯蓄する
func worker(
	ctx context.Context,
	vRepo *repository.VideoRepository,
	nRepo *repository.NotificationRepository,
	rRepo *repository.RSSRepository,
	cfg *config.Config,
) {
	vClient := client.NewVideoClient("https://snapshot.search.nicovideo.jp/api/v2/snapshot", fmt.Sprintf("nicovideo-rss-diy/%s service", cfg.System.Version))
	tClient := client.NewThumbnailClient(fmt.Sprintf("nicovideo-rss-diy/%s service", cfg.System.Version))

	// queriesに基づき動画検索を行いvRepoに追加する。クエリとクエリの間に1分待機する
	doVideo := func(
		ctx context.Context,
		vRepo *repository.VideoRepository,
		nRepo *repository.NotificationRepository,
		vClient *client.VideoClient,
		queries []config.SearchQuery,
		rangeStart time.Time,
		rangeEnd time.Time,
	) {
		// todo: そもそもクエリごとに知る限りの最新動画を覚えておけばもっと最適なAPIリクエストが可能。ただそれを誰に持たせるのかは考える必要がある

		slog.Debug(fmt.Sprintf("=== search start (%d queries)", len(queries)))
		slog.Debug(fmt.Sprintf("search startTime: %s", rangeStart.Format(time.RFC3339)))
	LOOP:
		for i, q := range queries {
			slog.Debug(fmt.Sprintf("  #%d q:%s", i, q.Query))
			searchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			reqBeginAt := time.Now()
			resp, err := vClient.SearchVideo(searchCtx, q.Query, []string{
				fmt.Sprintf("[startTime][lte]=%s", rangeStart.Format(time.RFC3339)),
				fmt.Sprintf("[startTime][gt]=%s", rangeEnd.Format(time.RFC3339))})
			reqEndAt := time.Now()
			cancel()
			if err != nil {
				slog.Error(err.Error())

				if errors.Is(err, client.ErrFiltersFormat) || errors.Is(err, client.ErrRespQueryParse) {
					nRepo.AddNotification(
						repository.NotificationError,
						"動画検索の際にエラーが発生しました。",
						err,
						true,
					)
					continue
				} else {
					nRepo.AddNotification(
						repository.NotificationError,
						"動画検索の際にエラーが発生しました。次回検索はクールダウン後になります。",
						err,
						true,
					)
					break LOOP
				}

			}

			vRepo.AddSortedVideos(resp.Videos)

			reqTime := reqEndAt.Sub(reqBeginAt)
			if i < len(queries)-1 {
				// API利用制限: 「繰り返しAPIリクエストを行う場合は、前回のAPIレスポンス時間と同じだけ待機時間を設けてご利用ください。」
				// 基本的に1分待つ。ただし念の為リクエストにそれ以上かかった場合はそれだけ待つ
				waitTime := 1 * time.Minute
				if reqTime > waitTime {
					waitTime = reqTime
				}
				slog.Debug(fmt.Sprintf("    req time: %d ms, total videos: %d, continue to next query: %.2f sec", reqTime.Milliseconds(), len(resp.Videos), waitTime.Seconds()))

				select {
				case <-ctx.Done():
					slog.Debug("worker(query): context done during wait between queries")
					return
				case <-time.After(waitTime):
					// continue searching
				}
			} else {
				slog.Debug(fmt.Sprintf("    req time: %d ms, total videos: %d", reqTime.Milliseconds(), len(resp.Videos)))
			}
		}
		slog.Debug("=== search end")

	}

	// vRepo内の動画を走査し、サムネイルのType, Lengthが未取得のものについて取得する。1件ごとに1秒待機する
	doThumbnail := func(
		ctx context.Context,
		vRepo *repository.VideoRepository,
		nRepo *repository.NotificationRepository,
		tClient *client.ThumbnailClient,
	) {
		// セマフォで同時接続上限をかけながら集めてもいいのかもしれないが
		// 踏ん切りがつかなかったために1秒間隔直列。200秒ならば許容範囲
		const ERROR_KEEPON_THRESHOLD = 5
		errorCount := 0

		slog.Debug(fmt.Sprintf("=== thumbnail start (%d videos(include already fetched))", len(vRepo.Videos)))
		waitMsSumForAvr := int64(0)
		thumbnailFetchedCountForAvr := int64(0)
		thumbnailFetchedCountTotal := int64(0)
	LOOP:
		for i, v := range vRepo.Videos {
			if v.ThumbnailType == "" || v.ThumbnailLength == 0 {
				thumbCtx, cancel := context.WithTimeout(ctx, 20*time.Second)

				reqBeginAt := time.Now()
				thumbMeta, err := tClient.FetchThumbnailMeta(thumbCtx, v.ThumbnailURL)
				reqEndAt := time.Now()
				cancel()
				if err != nil {
					slog.Error(err.Error())
					slog.Debug(fmt.Sprintf("%d", i))

					errorCount++
					if errorCount >= ERROR_KEEPON_THRESHOLD {
						nRepo.AddNotification(
							repository.NotificationError,
							"サムネイル画像情報取得の際に連続でエラーが発生しました。次回取得はクールダウン後になります。",
							err,
							true,
						)
						break LOOP
					}
					continue

				}

				errorCount = 0

				v.ThumbnailType = thumbMeta.Type
				v.ThumbnailLength = thumbMeta.Length

				thumbnailFetchedCountForAvr++
				thumbnailFetchedCountTotal++

				reqTime := reqEndAt.Sub(reqBeginAt)
				if i < len(vRepo.Videos)-1 {
					// API利用制限: 「繰り返しAPIリクエストを行う場合は、前回のAPIレスポンス時間と同じだけ待機時間を設けてご利用ください。」
					// CDNにも適用されるのか分からないが
					// 基本的に1秒待つ。ただし念の為リクエストにそれ以上かかった場合はそれだけ待つ
					waitTime := 1 * time.Second
					if reqTime > waitTime {
						waitTime = reqTime
					}
					waitMsSumForAvr += waitTime.Milliseconds()
					select {
					case <-ctx.Done():
						slog.Debug("worker(thumbnail): context done during wait between queries")
						return
					case <-time.After(waitTime):
						// continue fetching thumbnails
					}
				}
			}

			if (i+1)%50 == 0 {
				avr := float64(waitMsSumForAvr)
				if thumbnailFetchedCountForAvr > 0 { // thumbnailFetchedCount=0のときゼロ除算する
					avr = float64(waitMsSumForAvr) / float64(thumbnailFetchedCountForAvr)
				}
				slog.Debug(fmt.Sprintf("   %d. request cooldown average: %.2f msec.(%d requests)", i+1, avr, thumbnailFetchedCountForAvr))
				waitMsSumForAvr = 0
				thumbnailFetchedCountForAvr = 0
			}
		}

		slog.Debug(fmt.Sprintf("=== thumbnail end (total %d thumbnails metadata fetched)", thumbnailFetchedCountTotal))
	}

	GetNoNewDataLaterFallbackVal := func() time.Time { return time.Now() }

	// 15分毎に動画取得・サムネイル取得・RSS生成・貯蓄を繰り返す
	const LOOP_INTERVAL = 15 * time.Minute
	noNewDataLater := GetNoNewDataLaterFallbackVal()
	for {
		nRepo.ClearNotifications()

		searchStart := time.Now()
		searchStart = searchStart.AddDate(0, 0, -1) // APIが提供するデータは05:00時点。24時間ずれなければずっと05:00時点データで固まる
		searchEnd := searchStart.AddDate(-1, 0, 0)

		t := time.Now() // for debug output
		if searchStart.Before(noNewDataLater.Add(LOOP_INTERVAL)) {
			doVideo(ctx, vRepo, nRepo, vClient, cfg.SearchQueries, searchStart, searchEnd)
		} else {
			slog.Debug("### Update skipped, there are no new data")
			nRepo.AddNotification(
				repository.NotificationInfo,
				"更新を一時停止中...",
				errors.New("動画スナップショットAPIのデータ切り替えを待っています"),
				true,
			)
		}

		if cfg.VideoFetcher.ShouldFetchThumbnail {
			// 別に上のifへ入れてもいいが、サムネイル全部揃っているなら追加リクエストしないし
			// 前回ループでエラーなどで不足あれば取得した方が良いので
			doThumbnail(ctx, vRepo, nRepo, tClient)
		}
		// ↑によりfalseの場合サムネイルはURLだけ揃っている状態だが、
		// RSS生成部は全部データが揃っているときのみフィードにデータを加えるので問題ない

		lastModified, err := vClient.FetchLastModified(ctx)
		if err != nil {
			slog.Error(fmt.Sprintf("データ切り替え日時を取得できません: %v", err))
			noNewDataLater = GetNoNewDataLaterFallbackVal()
		} else {
			noNewDataLater = time.Date(lastModified.Year(), lastModified.Month(), lastModified.Day(), 5, 0, 0, 0, lastModified.Location())
		}

		slog.Debug(fmt.Sprintf("動画データは %s 時点まで存在", noNewDataLater.Format(time.RFC3339)))
		slog.Debug(fmt.Sprintf("### All done! (%s)", time.Since(t)))

		rssBytes, err := rss.GenerateRSS(nRepo.Notifications, vRepo.Videos, &cfg.RssGenerator)
		if err != nil {
			panic(fmt.Sprintf("RSSの生成に失敗しました: %v", err))
		}
		rRepo.SetFeed(rssBytes)

		select {
		case <-ctx.Done():
			slog.Debug("worker(loop): context done, exiting")
			return
		case <-time.After(LOOP_INTERVAL):
			// continue
		}
	}
}
