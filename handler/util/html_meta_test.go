package handler

import (
	"io"
	"testing"
)

type MockPage struct {
	Content string
	done    bool
}

func NewMockPage(s string) MockPage {
	return MockPage{Content: s}
}

func (mp *MockPage) Read(p []byte) (n int, err error) {
	if mp.done {
		return 0, io.EOF
	}
	for i, b := range []byte(mp.Content) {
		p[i] = b
	}
	mp.done = true
	return len(mp.Content), nil
}

func TestTitle(t *testing.T) {
	title := "foo bar"
	mp := NewMockPage("<html><head><title>" + title + "</title></head></html>")

	html_md := extractHTMLMetadata(&mp)

	if html_md.Title != title {
		t.Error("Expected title to be", title, ", but was:", html_md.Title)
	}
}

func TestDesc(t *testing.T) {
	desc := "foo bar"
	mp := NewMockPage("<html><head><meta property=\"description\" content=\"" + desc + "\"></head></html>")

	html_md := extractHTMLMetadata(&mp)

	if html_md.Desc != desc {
		t.Error("Expected desc to be", desc, ", but was:", html_md.Desc)
	}
}

func TestOGTitle(t *testing.T) {
	title := "foo bar"
	mp := NewMockPage("<html><head><meta property=\"og:title\" content=\"" + title + "\"></head></html>")

	html_md := extractHTMLMetadata(&mp)

	if html_md.OGTitle != title {
		t.Error("Expected og:title to be", title, ", but was:", html_md.OGTitle)
	}
}

func TestOGDesc(t *testing.T) {
	desc := "foo bar"
	mp := NewMockPage("<html><head><meta property=\"og:description\" content=\"" + desc + "\"></head></html>")

	html_md := extractHTMLMetadata(&mp)

	if html_md.OGDesc != desc {
		t.Error("Expected og:description to be", desc, ", but was:", html_md.OGDesc)
	}
}

func TestOGImage(t *testing.T) {
	image := "http://google.com/images/blah.jpg"
	mp := NewMockPage("<html><head><meta property=\"og:image\" content=\"" + image + "\"></head></html>")

	html_md := extractHTMLMetadata(&mp)

	if html_md.OGImage != image {
		t.Error("Expected og:image to be", image, ", but was:", html_md.OGImage)
	}
}

func TestOGAuthor(t *testing.T) {
	author := "someone"
	mp := NewMockPage("<html><head><meta property=\"og:author\" content=\"" + author + "\"></head></html>")

	html_md := extractHTMLMetadata(&mp)

	if html_md.OGAuthor != author {
		t.Error("Expected og:author to be", author, ", but was:", html_md.OGAuthor)
	}
}

func TestOGPublisher(t *testing.T) {
	publisher := "someone"
	mp := NewMockPage("<html><head><meta property=\"og:publisher\" content=\"" + publisher + "\"></head></html>")

	html_md := extractHTMLMetadata(&mp)

	if html_md.OGPublisher != publisher {
		t.Error("Expected og:publisher to be", publisher, ", but was:", html_md.OGPublisher)
	}
}

func TestOGSiteName(t *testing.T) {
	sitename := "Google"
	mp := NewMockPage("<html><head><meta property=\"og:site_name\" content=\"" + sitename + "\"></head></html>")

	html_md := extractHTMLMetadata(&mp)

	if html_md.OGSiteName != sitename {
		t.Error("Expected og:site_name to be", sitename, ", but was:", html_md.OGSiteName)
	}
}

func TestExtractHTMLMetadata(t *testing.T) {
	title := "foobar"
	description := "boo far"
	og_title := "Foo Bar"
	og_desc := "Boo Far"
	og_image := "http://google.com/images/blah.jpg"
	og_author := "someone"
	og_publisher := "someone else"
	og_sitename := "Google"
	twitter_title := "Twitter Title"
	twitter_desc := "Twitter Description"
	twitter_image := "http://twitter.com/images/blah.jpg"

	mp := NewMockPage(`
	<html>
		<head>
			<title>` + title + `</title>
			<meta property="description" content="` + description + `">
			<meta property="og:title" content="` + og_title + `">
			<meta property="og:description" content="` + og_desc + `">
			<meta property="og:image" content="` + og_image + `">
			<meta property="og:author" content="` + og_author + `">
			<meta property="og:publisher" content="` + og_publisher + `">
			<meta property="og:site_name" content="` + og_sitename + `">
			<meta property="twitter:title" content="` + twitter_title + `">
			<meta property="twitter:description" content="` + twitter_desc + `">
			<meta property="twitter:image" content="` + twitter_image + `">
		</head>
	</html>`)

	html_md := extractHTMLMetadata(&mp)

	if html_md.Desc != description {
		t.Error("Expected description to be", description, ", but was:", html_md.Desc)
	}
	if html_md.OGTitle != og_title {
		t.Error("Expected og:title to be", og_title, ", but was:", html_md.OGTitle)
	}
	if html_md.OGDesc != og_desc {
		t.Error("Expected og:description to be", og_desc, ", but was:", html_md.OGDesc)
	}
	if html_md.OGImage != og_image {
		t.Error("Expected og:image to be", og_image, ", but was:", html_md.OGImage)
	}
	if html_md.OGAuthor != og_author {
		t.Error("Expected og:author to be", og_author, ", but was:", html_md.OGAuthor)
	}
	if html_md.OGPublisher != og_publisher {
		t.Error("Expected og:publisher to be", og_publisher, ", but was:", html_md.OGSiteName)
	}
	if html_md.OGSiteName != og_sitename {
		t.Error("Expected og:site_name to be", og_sitename, ", but was:", html_md.OGSiteName)
	}
	if html_md.TwitterTitle != twitter_title {
		t.Error("Expected twitter:title to be", twitter_title, ", but was:", html_md.TwitterTitle)
	}
	if html_md.TwitterDesc != twitter_desc {
		t.Error("Expected twitter:description to be", twitter_desc, ", but was:", html_md.TwitterDesc)
	}
	if html_md.TwitterImage != twitter_image {
		t.Error("Expected twitter:image to be", twitter_image, ", but was:", html_md.TwitterImage)
	}
}
