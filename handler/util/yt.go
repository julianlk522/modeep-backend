package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"
)

const YT_VID_URL_REGEX = `^(https?://)?(www\.)?(youtube\.com|youtu\.be)/.*`

// helpers for YouTube video links
func IsYouTubeVideoLink(url string) bool {
	if !strings.Contains(url, "youtube.com/watch?v=") && !strings.Contains(url, "youtu.be/") {
		return false
	}

	// prevent links containing YouTube URLs
	match, _ := regexp.MatchString(YT_VID_URL_REGEX, url)
	return match

}

func ObtainYouTubeMetaData(request *model.NewLinkRequest) error {
	id := ExtractYouTubeVideoID(request.NewLink.URL)
	if id == "" {
		return e.ErrInvalidURL
	}

	API_KEY := os.Getenv("FITM_GOOGLE_API_KEY")
	if API_KEY == "" {
		log.Printf("Could not find API_KEY")
		return e.ErrGoogleAPIsKeyNotFound
	}

	gAPIs_url := "https://www.googleapis.com/youtube/v3/videos?id=" + id + "&key=" + API_KEY + "&part=snippet"

	resp, err := http.Get(gAPIs_url)
	if err != nil {
		log.Print(e.ErrGoogleAPIsRequestFail(err))
		return e.ErrGoogleAPIsRequestFail(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = e.ErrInvalidGoogleAPIsResponse(resp.Status)
		log.Print(err.Error())
		return err
	}

	video_data, err := ExtractMetaDataFromGoogleAPIsResponse(resp.Body)
	if err != nil {
		return err
	}

	request.AutoSummary = video_data.Items[0].Snippet.Title
	request.ImgURL = video_data.Items[0].Snippet.Thumbnails.Default.URL
	request.URL = "https://www.youtube.com/watch?v=" + id

	// no errors
	return nil
}

func ExtractYouTubeVideoID(url string) string {
	if strings.Contains(url, "youtube.com/watch?v=") {
		return strings.Split(strings.Split(url, "&")[0], "?v=")[1]
	} else if strings.Contains(url, "youtu.be/") {
		return strings.Split(strings.Split(url, "youtu.be/")[1], "?")[0]
	}

	return ""
}

func ExtractMetaDataFromGoogleAPIsResponse(body io.Reader) (model.YTVideoMetaData, error) {
	var meta model.YTVideoMetaData

	err := json.NewDecoder(body).Decode(&meta)
	if err != nil {
		err = e.ErrGoogleAPIsResponseExtractionFail(err)
		log.Print(err)
	}
	return meta, err
}
