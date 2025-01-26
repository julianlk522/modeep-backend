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

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.Title != title {
		t.Error("Expected title to be", title, ", but was:", html_md.Title)
	}
}

func TestDescription(t *testing.T) {
	desc := "foo bar"
	mp := NewMockPage("<html><head><meta property=\"description\" content=\"" + desc + "\"></head></html>")

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.Description != desc {
		t.Error("Expected desc to be", desc, ", but was:", html_md.Description)
	}
}

func TestOGTitle(t *testing.T) {
	title := "foo bar"
	mp := NewMockPage("<html><head><meta property=\"og:title\" content=\"" + title + "\"></head></html>")

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.OGTitle != title {
		t.Error("Expected og:title to be", title, ", but was:", html_md.OGTitle)
	}
}

func TestOGDescription(t *testing.T) {
	desc := "foo bar"
	mp := NewMockPage("<html><head><meta property=\"og:description\" content=\"" + desc + "\"></head></html>")

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.OGDescription != desc {
		t.Error("Expected og:description to be", desc, ", but was:", html_md.OGDescription)
	}
}

func TestOGImage(t *testing.T) {
	image := "http://google.com/images/blah.jpg"
	mp := NewMockPage("<html><head><meta property=\"og:image\" content=\"" + image + "\"></head></html>")

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.OGImage != image {
		t.Error("Expected og:image to be", image, ", but was:", html_md.OGImage)
	}
}

func TestOGAuthor(t *testing.T) {
	author := "jonlaing"
	mp := NewMockPage("<html><head><meta property=\"og:author\" content=\"" + author + "\"></head></html>")

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.OGAuthor != author {
		t.Error("Expected og:author to be", author, ", but was:", html_md.OGAuthor)
	}
}

func TestOGPublisher(t *testing.T) {
	publisher := "jonlaing"
	mp := NewMockPage("<html><head><meta property=\"og:publisher\" content=\"" + publisher + "\"></head></html>")

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.OGPublisher != publisher {
		t.Error("Expected og:publisher to be", publisher, ", but was:", html_md.OGPublisher)
	}
}

func TestOGSiteName(t *testing.T) {
	sitename := "Google"
	mp := NewMockPage("<html><head><meta property=\"og:site_name\" content=\"" + sitename + "\"></head></html>")

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.OGSiteName != sitename {
		t.Error("Expected og:site_name to be", sitename, ", but was:", html_md.OGSiteName)
	}
}

func TestExtractHTMLMetadata(t *testing.T) {
	title := "foobar"
	description := "boo far"
	ogTitle := "Foo Bar"
	ogDesc := "Boo Far"
	ogImage := "http://google.com/images/blah.jpg"
	ogAuthor := "Jon Laing"
	ogPublisher := "jonlaing"
	ogSiteName := "Google"

	mp := NewMockPage(`
	<html>
		<head>
			<title>` + title + `</title>
			<meta property="description" content="` + description + `">
			<meta property="og:title" content="` + ogTitle + `">
			<meta property="og:description" content="` + ogDesc + `">
			<meta property="og:image" content="` + ogImage + `">
			<meta property="og:author" content="` + ogAuthor + `">
			<meta property="og:publisher" content="` + ogPublisher + `">
			<meta property="og:site_name" content="` + ogSiteName + `">
		</head>
	</html>`)

	html_md := ExtractHTMLMetadata(&mp)

	if html_md.Description != description {
		t.Error("Expected description to be", description, ", but was:", html_md.Description)
	}
	if html_md.OGTitle != ogTitle {
		t.Error("Expected og:title to be", ogTitle, ", but was:", html_md.OGTitle)
	}
	if html_md.OGDescription != ogDesc {
		t.Error("Expected og:description to be", ogDesc, ", but was:", html_md.OGDescription)
	}
	if html_md.OGImage != ogImage {
		t.Error("Expected og:image to be", ogImage, ", but was:", html_md.OGImage)
	}
	if html_md.OGAuthor != ogAuthor {
		t.Error("Expected og:author to be", ogAuthor, ", but was:", html_md.OGAuthor)
	}
	if html_md.OGPublisher != ogPublisher {
		t.Error("Expected og:publisher to be", ogPublisher, ", but was:", html_md.OGSiteName)
	}
	if html_md.OGSiteName != ogSiteName {
		t.Error("Expected og:site_name to be", ogSiteName, ", but was:", html_md.OGSiteName)
	}
}
