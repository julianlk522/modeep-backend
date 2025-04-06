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

const YT_VID_URL_REGEX = `^(https?:\/\/)?(www\.)?(youtube\.com|youtu\.be)\/.+`

func IsYTVideo(url string) bool {
	match, _ := regexp.MatchString(YT_VID_URL_REGEX, url)
	return match

}

func GetYTVideoMetadata(url string) (*model.YTVideoMetadata, error) {
	id := ExtractYTVideoID(url)
	if id == "" {
		return nil, e.ErrInvalidURL
	}

	API_KEY := os.Getenv("FITM_GOOGLE_API_KEY")
	if API_KEY == "" {
		log.Printf("Could not find API_KEY")
		return nil, e.ErrGoogleAPIsKeyNotFound
	}

	gAPIs_url := "https://www.googleapis.com/youtube/v3/videos?id=" + id + "&key=" + API_KEY + "&part=snippet"

	resp, err := http.Get(gAPIs_url)
	if err != nil {
		log.Print(e.ErrGoogleAPIsRequestFail(err))
		return nil, e.ErrGoogleAPIsRequestFail(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = e.ErrInvalidGoogleAPIsResponse(resp.Status)
		log.Print(err.Error())
		return nil, err
	}

	yt_md, err := ExtractGoogleAPIsResponseMetadata(resp.Body)
	if err != nil {
		return nil, err
	}

	yt_md.ID = id

	return yt_md, nil
}

func ExtractYTVideoID(url string) string {
	if strings.Contains(url, "youtube.com/watch?v=") {
		return strings.Split(strings.Split(url, "&")[0], "?v=")[1]
	} else if strings.Contains(url, "youtu.be/") {
		return strings.Split(strings.Split(url, "youtu.be/")[1], "?")[0]
	}

	return ""
}

func ExtractGoogleAPIsResponseMetadata(body io.Reader) (*model.YTVideoMetadata, error) {
	var yt_md model.YTVideoMetadata

	err := json.NewDecoder(body).Decode(&yt_md)
	if err != nil {
		err = e.ErrGoogleAPIsResponseExtractionFail(err)
		log.Print(err)
	}
	return &yt_md, err
}
