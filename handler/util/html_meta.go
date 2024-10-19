package handler

import (
	"io"

	"golang.org/x/net/html"
)

type HTMLMeta struct {
	Title         string
	Description   string
	OGTitle       string
	OGDescription string
	OGImage       string
	OGAuthor      string
	OGPublisher   string
	OGSiteName    string
}

func ExtractMetaFromHTMLTokens(resp io.Reader) (hm HTMLMeta) {
	z := html.NewTokenizer(resp)

	title_tag := false
	title_found := false

	for {
		token_type := z.Next()
		switch token_type {
		case html.ErrorToken:
			return
		case html.SelfClosingTagToken, html.StartTagToken:
			t := z.Token()
			if t.Data == "title" && !title_found {
				title_tag = true
			} else if t.Data == "meta" {
				AssignTokenPropertyToHTMLMeta(t, &hm)
			}
		case html.TextToken:
			if title_tag {
				t := z.Token()
				hm.Title = t.Data

				title_tag = false
				title_found = true
			}
		}
	}
}

var meta_properties = []string{
	"description",
	"og:title",
	"og:description",
	"og:image",
	"og:author",
	"og:publisher",
	"og:site_name",
}

func AssignTokenPropertyToHTMLMeta(t html.Token, hm *HTMLMeta) {
	for _, prop := range meta_properties {
		m, ok := ExtractMetaProperty(t, prop)
		if ok {
			switch prop {
			case "description":
				hm.Description = m
			case "og:title":
				hm.OGTitle = m
			case "og:description":
				hm.OGDescription = m
			case "og:image":
				hm.OGImage = m
			case "og:author":
				hm.OGAuthor = m
			case "og:publisher":
				hm.OGPublisher = m
			case "og:site_name":
				hm.OGSiteName = m
			}
		}
	}
}

func ExtractMetaProperty(t html.Token, prop string) (content string, ok bool) {
	for _, attr := range t.Attr {
		if (attr.Key == "property" || attr.Key == "name") && attr.Val == prop {
			ok = true
		}

		if attr.Key == "content" {
			content = attr.Val
		}
	}

	return
}
