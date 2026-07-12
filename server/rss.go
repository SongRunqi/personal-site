package main

import (
	"encoding/xml"
	"log"
	"net/http"
	"time"
)

const rssLimit = 20

type rssFeed struct {
	XMLName      xml.Name   `xml:"rss"`
	Version      string     `xml:"version,attr"`
	ContentXMLNS string     `xml:"xmlns:content,attr"`
	Channel      rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        string   `xml:"guid"`
	PubDate     string   `xml:"pubDate"`
	Description string   `xml:"description"`
	Content     rssCDATA `xml:"content:encoded"`
}

type rssCDATA struct {
	Text string `xml:",cdata"`
}

func (s *server) handleFeed(w http.ResponseWriter, r *http.Request) {
	posts := s.mergedFullPosts()
	if len(posts) > rssLimit {
		posts = posts[:rssLimit]
	}

	items := make([]rssItem, 0, len(posts))
	for _, p := range posts {
		link := s.baseURL + "/blog/" + p.Slug
		items = append(items, rssItem{
			Title:       p.Title,
			Link:        link,
			GUID:        link,
			PubDate:     p.Date.Format(time.RFC1123Z),
			Description: p.Summary,
			Content:     rssCDATA{Text: p.HTML},
		})
	}

	feed := rssFeed{
		Version:      "2.0",
		ContentXMLNS: "http://purl.org/rss/1.0/modules/content/",
		Channel: rssChannel{
			Title:       "宋一天的博客",
			Link:        s.baseURL,
			Description: "独立开发者宋一天的项目与写作",
			Language:    "zh-cn",
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		log.Printf("RSS 编码失败:%v", err)
	}
}
