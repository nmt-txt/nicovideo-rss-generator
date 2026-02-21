# Nicovideo RSS Generator

提供終了したニコニコ動画新着動画RSSを擬似的に復活させるサーバーソフトウェア。  
ニコニコ動画のAPIから設定された条件で動画一覧を取得し、RSSフィードを生成・提供する。更新間隔は15分である。  

Docker Composeを用いて稼働させることを想定している。

<details>

<summary>生成されるRSSのサンプル</summary>

```xml
<rss version="2.0">
  <channel>
    <title>Nicovideo RSS DIY</title>
    <link>https://www.nicovideo.jp/</link>
    <description>ニコニコ動画新着RSS(自作)</description>
    <item>
      <title>動画タイトル</title>
      <link>視聴URL</link>
      <description>動画説明文(XML特殊文字エスケープ有り)</description>
      <pubDate>投稿日時(RFC 822)</pubDate>
      <guid isPermaLink="true">視聴URL</guid>
      <enclosure url="サムネイルURL(shouldFetchThumbnail=true時のみ存在)" length="サムネイル画像サイズ" type="image/jpeg"></enclosure>
      <category domain="タグ検索URL">タグ名</category>
      <category domain="タグ検索URL2">タグ名2</category>
    </item>
    <item>
      <title>新・豪血寺一族 -煩悩解放 - レッツゴー！陰陽師</title>
      <link>https://nico.ms/sm9</link>
      <description>レッツゴー！陰陽師（フルコーラスバージョン）</description>
      <pubDate>06 Mar 07 00:33 JST</pubDate>
      <guid isPermaLink="true">https://nico.ms/sm9</guid>
      <enclosure url="https://nicovideo.cdn.nimg.jp/thumbnails/9/9" length="6337" type="image/jpeg"></enclosure>
      <category domain="https://www.nicovideo.jp/tag/%E9%99%B0%E9%99%BD%E5%B8%AB">陰陽師</category>
      <category domain="https://www.nicovideo.jp/tag/%E3%83%AC%E3%83%83%E3%83%84%E3%82%B4%E3%83%BC%EF%BC%81%E9%99%B0%E9%99%BD%E5%B8%AB">レッツゴー！陰陽師</category>
      <category domain="https://www.nicovideo.jp/tag/%E5%85%AC%E5%BC%8F">公式</category>
      <category domain="https://www.nicovideo.jp/tag/%E9%9F%B3%E6%A5%BD">音楽</category>
      <category domain="https://www.nicovideo.jp/tag/%E3%82%B2%E3%83%BC%E3%83%A0">ゲーム</category>
      <category domain="https://www.nicovideo.jp/tag/%E5%BC%BE%E5%B9%95%E5%8B%95%E7%94%BB">弾幕動画</category>
      <category domain="https://www.nicovideo.jp/tag/%E4%BC%9D%E8%AA%AC">伝説</category>
      <category domain="https://www.nicovideo.jp/tag/%E6%9C%80%E5%8F%A4%E3%81%AE%E5%8B%95%E7%94%BB">最古の動画</category>
      <category domain="https://www.nicovideo.jp/tag/3%E6%9C%886%E6%97%A5%E6%8A%95%E7%A8%BF%E5%8B%95%E7%94%BB">3月6日投稿動画</category>
      <category domain="https://www.nicovideo.jp/tag/%E9%87%8D%E8%A6%81%E3%83%8B%E3%82%B3%E3%83%8B%E3%82%B3%E6%96%87%E5%8C%96%E8%B2%A1">重要ニコニコ文化財</category>
      <category domain="https://www.nicovideo.jp/tag/sm9">sm9</category>
    </item>
  </channel>
</rss>
```

</details>

## 設定

config.jsonに記述し、docker-compose.yml内で`/config/config.json`へとバインドマウントする。  
設定ファイルを変更した場合、ソフトウェアの再起動が必要である。

記述例:

```json
{
    "searchQueries": [
        {"query": "VOCALOID"},
        {"query": "ソフトウェアトーク車載 OR ソフトウェアトーク旅行"},
    ],
    "videoFetcher": {
        "shouldFetchThumbnail": false
    }
}
```

この場合`VOCALOID` と `ソフトウェアトーク車載 OR ソフトウェアトーク旅行`の2つで検索を行い結果を混ぜた上で、最新順200件をフィードに表示する。検索タグの数に上限はないものの1つ増やせば1分[更新作業が長くなる](#制限)(おそらくフィードが空な起動直後しか気にならないと思われるが)。  
画像を表示できるRSSリーダーの場合、`shouldFetchThumbnail`の後にある`false`を`true`に書き換えることで動画サムネイル情報が付与されるようになる。  

<details>

<summary>その他の設定項目</summary>

サンプル:

```json
{
    "searchQueries": [
        {"query": "VOCALOID OR SynthesizerV OR NEUTRINO OR ソフトウェアシンガー"}
    ],
    "log": "debug",
    "rssGenerator": {
        "title": "カスタムRSSタイトル",
        "link": "https://カスタムRSS-link-URL.net",
        "description": "カスタムRSS description",
        "shouldSuppressSystemMessage": true
    },
    "videoFetcher": {
        "shouldFetchThumbnail": true
    }
}

```

- `log`(string, 省略時:info)
  - ログレベルを指定する。error > info > debugで、指定したもの以上のログのみ出力する
  - 例:infoを指定した場合、errorとinfoのみ出力する
- `rssGenerator`(省略可能)
  - RSS生成部にかかわる設定を行う。ブロックごと省略可能である。ほとんどはRSSの要素をデフォルト値から変更するためにある。省略時の値は上部「生成されるRSSのサンプル」内を参照
  - `title`(string)
  - `link`(string)
  - `description`(string)
  - `suppressSystemMessage`(bool)
    - 「\[INFO\]サーバーの更新を待っています」のようなアプリケーション由来のメッセージをRSSへ出力する機能を無効にする
    - RSSを人が読む場合はおおよそメリットのある機能だと思われるが、RSSを更に機械で処理する場合などは邪魔になるため、無効化する方が良いだろう
- `videoFetcher`(省略可能)
  - 動画情報取得部にかかわる設定を行う。ブロックごと省略可能である。
  - `shouldFetchThumbnail`(bool, 省略時:false)
    - 動画サムネイル情報を取得するかどうかを指定する
    - 全てのリーダーがenclosureの画像を表示するわけではない。必要な場合にのみ取得することで、余計な処理時間とリクエストを削減する
  
</details>

## 起動・終了

起動: `$ docker compose up -d`  
RSSフィードは`/`で取得できる。(例:`http://[マシンIPアドレス]:2525/`)

終了: `$ docker compose down`

## 制限

- フィードに載る動画は最大200件まで
- タグ完全一致検索で一致したもののみフィードに載る
- フィードの更新は15分ごとに開始される。更新には以下の時間が必要となり全て完了してからRSSが書き換わる
  - 検索1件ごとに1分 (検索APIリクエスト間隔)
  - 動画1本ごとに1秒 (サムネイル情報リクエスト間隔、取得済みは除外のため最大200秒,最小0秒)
  - 各リクエストが返ってくるまでの時間
- 動画は**1日前**～1年前のものに限られる
  - 投稿されてからフィードに掲載されるまで24時間の遅れがある
  - 利用するAPIのデータは05:00時点のもので固定されリアルタイムに更新されないため([後述](#フィードに載る動画について))

### フィードに載る動画について

利用する[スナップショット検索API v2](https://site.nicovideo.jp/search-api-docs/snapshot)が提供するデータの制限として:

- データは1日1回更新
  - リアルタイムではない
- 参照できるデータはAM 5:00時点のもの
  - つまり今日のAM 5:00までのデータしか存在しない(日付を越した場合は前日のAM 5:00)
  - 実際のところそのデータが得られるようになるのは約2時間後のAM 7:00頃のようである

がある。  これのために本ソフトウェアがAPIから取得できる動画は05:00までのものに限られ、リアルタイムに最新動画を載せることは不可能である。  
そのため意図的に24時間前時点の最新動画データを取得・RSS生成することでフィードがリアルタイムに更新されているかのように見せかけている。最新動画情報を文字通りリアルタイムで得たいという用途でこのソフトウェアを利用することは**できない**。  

また、前述の通りデータは05:00までだがそのデータが利用できるようになるのは07:00頃のようであるため5時から7時頃まではフィードが更新されない。

```plaintext
例: 18日の下記各時刻に本ソフトがフィードを生成する場合(24時間表記)
#           [データは17日05:00までが存在]
- 18日04:00 -> 17日04:00までのデータで生成
- 18日05:00 -> 17日05:00まで
- 18日06:00 -> 17日05:00まで ←6時のデータはまだ存在しない
# 18日07:00 [データ更新!、データは18日05:00までが存在]
- 18日07:00 -> 17日07:00まで
```

## 不足

- API検索時のfiltersの指定を可能に
  - 特にjsonFilterにより投稿者を指定したフィードも可能になる
    - フォロー上限人数を超えた場合、RSSで疑似フォローを行う手法があったようなので
- フィードへの通知を抑制するトグル
  - エラーが発生した場合などはフィードへ通知を流すようにしているが場合によっては不便かもしれない
- RSSのタイトルなどをjsonから設定できるように
- コンフィグのホットリロード

## ビルド

外部パッケージを使用していないため  
`$ go build main.go`  

ただしdocker-composeを使用する場合は、ビルド操作がDockerfile内で既に記述されているため手動ビルドは不要である。  
いつも通りコード変更後は`--build`オプションを忘れずに。

イメージビルド＆実行: `$ docker compose up -d --build`
ビルドのみ: `docker build -t nicovideo-rss-diy ./`

---

`go build`によって.exeなどのバイナリファイルを生成し、Dockerを使用せず直接起動し利用することも可能なはずである。  
その際は起動時の引数としてconfigファイルを配置するディレクトリの絶対パスを与えることが必要となる。  
例: `.\main.exe "C:\Users\[WIN_USER_NAME]\Desktop\nrd"` (デスクトップ内nrdフォルダ内にconfig.jsonを配置する場合)
