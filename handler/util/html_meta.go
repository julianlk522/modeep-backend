package handler

import (
	"io"

	"golang.org/x/net/html"
)

type HTMLMetadata struct {
	Title         string
	Desc          string
	OGTitle       string
	OGDesc        string
	OGImage       string
	OGAuthor      string
	OGPublisher   string
	OGSiteName    string
	TwitterTitle  string
	TwitterDesc   string
	TwitterImage  string
}

func ExtractHTMLMetadata(resp io.Reader) (html_md HTMLMetadata) {
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
					AssignTokenPropertyToHTMLMeta(t, &html_md)
				}
			case html.TextToken:
				if title_tag {
					t := z.Token()
					html_md.Title = t.Data

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
	"twitter:title",
	"twitter:description",
	"twitter:image",
}

func AssignTokenPropertyToHTMLMeta(token html.Token, html_md *HTMLMetadata) {
	for _, mp := range meta_properties {
		prop, ok := ExtractMetaPropertyFromToken(mp, token)
		if ok {
			switch mp {
				case "description":
					html_md.Desc = prop
				case "og:title":
					html_md.OGTitle = prop
				case "og:description":
					html_md.OGDesc = prop
				case "og:image":
					html_md.OGImage = prop
				case "og:author":
					html_md.OGAuthor = prop
				case "og:publisher":
					html_md.OGPublisher = prop
				case "og:site_name":
					html_md.OGSiteName = prop
				case "twitter:title":
					html_md.TwitterTitle = prop
				case "twitter:description":
					html_md.TwitterDesc = prop
				case "twitter:image":
					html_md.TwitterImage = prop
			}
		}
	}
}

func ExtractMetaPropertyFromToken(mp string, token html.Token) (content string, ok bool) {
	for _, attr := range token.Attr {
		if (attr.Key == "property" || attr.Key == "name") && attr.Val == mp {
			ok = true
		}

		if attr.Key == "content" {
			content = attr.Val
		}
	}

	return
}
