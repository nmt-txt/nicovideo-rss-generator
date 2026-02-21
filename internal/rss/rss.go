package rss

import (
	"encoding/xml"
	"fmt"
	"nicovideoRSSDIY/internal/config"
	"nicovideoRSSDIY/internal/repository"
	"time"
)

// todo: interfaceを使えば、videoとnotificationで構造体を分けることも可能かもしれない?

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}
type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}
type Item struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link,omitempty"`
	Description string     `xml:"description"`
	PubDate     string     `xml:"pubDate,omitempty"`
	GUID        GUID       `xml:"guid"`
	Enclosure   *Enclosure `xml:"enclosure,omitempty"`
	Category    []Category `xml:"category,omitempty"`
}
type GUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink bool   `xml:"isPermaLink,attr"`
}
type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}
type Category struct {
	Value  string `xml:",chardata"`
	Domain string `xml:"domain,attr,omitempty"`
}

func GenerateRSS(
	notifications []repository.Notification,
	videos []*repository.Video,
	rssConfig *config.RssGenerator,
) ([]byte, error) {
	items := make([]Item, 0, len(notifications)+len(videos))

	if !rssConfig.ShouldSuppressSystemMessage {
		for _, n := range notifications {
			desc := n.Description.Error()
			if n.AllowDuplication && n.DuplicateCount > 0 {
				desc += fmt.Sprintf("(重複: %d件)", n.DuplicateCount+1)
			}

			title := fmt.Sprintf("[%s] %s", n.Level.String(), n.Title)

			guid := GUID{
				Value:       fmt.Sprintf("%d-%s", n.Date.Unix(), title),
				IsPermaLink: false,
			}

			items = append(items, Item{
				Title:       title,
				Description: desc,
				PubDate:     n.Date.Format(time.RFC822),
				GUID:        guid,
				Enclosure:   nil,
				Category:    nil,
			})
		}
	}

	for _, v := range videos {
		item := Item{
			Title:       v.Title,
			Link:        v.URL(),
			Description: v.Description,
			PubDate:     v.StartTime.Format(time.RFC822),
			GUID: GUID{
				Value:       v.URL(),
				IsPermaLink: true,
			},
			Enclosure: nil,
		}

		// あればサムネイルを付与
		if v.ThumbnailURL != "" && v.ThumbnailType != "" && v.ThumbnailLength > 0 {
			item.Enclosure = &Enclosure{
				URL:    v.ThumbnailURL,
				Type:   v.ThumbnailType,
				Length: v.ThumbnailLength,
			}
		}

		// あればタグをcategoryとして付与
		if v.TagsConnectedStr != "" {
			tags := v.Tags()
			categories := make([]Category, 0, len(tags))
			for _, tag := range tags {
				categories = append(categories, Category{
					Value:  tag,
					Domain: repository.TagSearchURL(tag),
				})
			}
			item.Category = categories
		}
		items = append(items, item)
	}

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       rssConfig.Title,
			Link:        rssConfig.Link,
			Description: rssConfig.Description,
			Items:       items,
		},
	}

	result, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("RSSの生成に失敗しました: %w", err)
	}
	return result, nil
}
