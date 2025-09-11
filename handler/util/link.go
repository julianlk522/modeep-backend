package handler

import (
	"crypto/tls"
	"log"
	"os"
	"slices"
	"strings"

	_ "golang.org/x/image/webp"

	"github.com/julianlk522/modeep/db"
	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"

	"database/sql"
	"fmt"

	"net/http"
)

// capitalized so it can be exported and used in GetPreviewImg handler
var Preview_img_dir string

const (
	MAX_DAILY_LINKS          = 50
	MAX_PREVIEW_IMG_WIDTH_PX = 200
)

func init() {
	test_data_path := os.Getenv("MODEEP_TEST_DATA_PATH")
	if test_data_path == "" {
		log.Panic("$MODEEP_TEST_DATA_PATH not set")
	}
	Preview_img_dir = test_data_path + "/img/preview"
}

func UserHasSubmittedMaxDailyLinks(login_name string) (bool, error) {
	var count int
	err := db.Client.QueryRow(`SELECT count(*)
		FROM Links
		WHERE submitted_by = ?
		AND submit_date >= date('now', '-1 days');`,
		login_name,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	return count >= MAX_DAILY_LINKS, nil
}

func PrepareLinksPage[T model.HasCats](links_sql *query.TopLinks, options *model.LinksPageOptions) (*model.LinksPage[T], error) {
	links_page, err := ScanRawLinksPageData[T](links_sql)
	if err != nil {
		return nil, err
	}
	
	PaginateLinks(links_page.Links)

	cats_params := options.Cats
	if cats_params != "" {
		CountMergedCatSpellingVariants(links_page, cats_params)
	}
	
	nsfw_params := options.NSFW
	hidden_links, err := GetNSFWLinksCount[T](links_sql, nsfw_params)
	if err != nil {
		return nil, err
	}

	links_page.NSFWLinks = hidden_links
	

	return links_page, nil
}

func ScanRawLinksPageData[T model.Link | model.LinkSignedIn](links_sql *query.TopLinks) (*model.LinksPage[T], error) {
	if links_sql.Error != nil {
		return nil, links_sql.Error
	}

	rows, err := db.Client.Query(links_sql.Text, links_sql.Args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	// NOTE: this both scans the links and creates the LinksPage struct
	// because number of pages must be taken here from the query result rows
	var links any
	var pages int

	switch any(new(T)).(type) {
	case *model.Link:
		var signed_out_links = []model.Link{}

		for rows.Next() {
			l := model.Link{}
			err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.Summary,
				&l.SummaryCount,
				&l.TimesStarred,
				&l.EarliestStarrers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,
				&pages,
			)
			if err != nil {
				return nil, err
			}
			signed_out_links = append(signed_out_links, l)
		}

		links = &signed_out_links

	case *model.LinkSignedIn:
		var signed_in_links = []model.LinkSignedIn{}

		for rows.Next() {
			l := model.LinkSignedIn{}
			if err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.Summary,
				&l.SummaryCount,
				&l.TimesStarred,
				&l.EarliestStarrers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,
				&pages,
				&l.StarsAssigned,
			); err != nil {
				return nil, err
			}

			signed_in_links = append(signed_in_links, l)
		}

		links = &signed_in_links
	}

	lp := &model.LinksPage[T]{Pages: -1}
	if links != nil && len(*links.(*[]T)) > 0 {
		lp.Links = links.(*[]T)
		lp.Pages = pages
	}

	return lp, nil
}

func ScanSingleLink[T model.Link | model.LinkSignedIn](single_link_sql *query.SingleLink) (*T, error) {
	var link any

	switch any(new(T)).(type) {
	case *model.LinkSignedIn:
		var l = &model.LinkSignedIn{}
		if err := db.Client.
			QueryRow(single_link_sql.Text, single_link_sql.Args...).
			Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.Summary,
				&l.SummaryCount,
				&l.TimesStarred,
				&l.EarliestStarrers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,
				&l.StarsAssigned,
			); err != nil {
			return nil, err
		}

		link = l
	case *model.Link:
		var l = &model.Link{}
		if err := db.Client.
			QueryRow(single_link_sql.Text, single_link_sql.Args...).
			Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.Summary,
				&l.SummaryCount,
				&l.TimesStarred,
				&l.EarliestStarrers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,
			); err != nil {
			return nil, err
		}

		link = l
	}

	return link.(*T), nil
}

func PaginateLinks[T model.LinkSignedIn | model.Link](links *[]T) {
	if links == nil || len(*links) == 0 {
		return
	} else if len(*links) == query.LINKS_PAGE_LIMIT+1 {
		*links = (*links)[0:query.LINKS_PAGE_LIMIT]
	}
}

func CountMergedCatSpellingVariants[T model.HasCats](lp *model.LinksPage[T], cats_params string) {
	if lp.Links == nil || len(*lp.Links) == 0 {
		return
	}

	cat_filters := strings.Split(strings.ToLower(cats_params), ",")

	for _, link := range *lp.Links {
		link_cats := strings.Split(link.GetCats(), ",")
		lc_link_cats := strings.Split(strings.ToLower(link.GetCats()), ",")

		for c, cat := range lc_link_cats {
			if !slices.Contains(cat_filters, cat) {
				for _, cf := range cat_filters {
					// preserve original casing
					if CatsAreSingularOrPluralVariationsOfEachOther(cat, cf) && !slices.Contains(lp.MergedCats, link_cats[c]) {
						lp.MergedCats = append(lp.MergedCats, link_cats[c])
					}
				}
			}
		}
	}
}

func GetNSFWLinksCount[T model.HasCats](links_sql *query.TopLinks, nsfw_params bool) (int, error) {
	hidden_links_count_sql := links_sql.CountNSFWLinks(nsfw_params)
	var hidden_links sql.NullInt32
	if err := db.Client.QueryRow(
		hidden_links_count_sql.Text, 
		hidden_links_count_sql.Args...,
	).Scan(&hidden_links); err != nil {
		return 0, err
	}
	
	return int(hidden_links.Int32), nil
}

// Add link (non-YT)
func GetLinkExtraMetadataFromResponse(resp *http.Response) *model.LinkExtraMetadata {
	if resp == nil {
		return nil
	} else if resp.StatusCode != http.StatusForbidden {
		html_md := ExtractHTMLMetadata(resp.Body)
		return GetLinkExtraMetadataFromHTML(html_md)
	}

	return nil
}

func GetResolvedURLResponse(url string) (*http.Response, error) {
	protocols := []string{"", "https://", "http://"}

	for _, p := range protocols {
		full_url := p + url
		req, err := http.NewRequest("GET", full_url, nil)
		if err != nil {
			continue
		}

		req.Header.Set("Accept", "*/*")
		req.Header.Set("User-Agent", "Modeep-Bot (https://modeep.org/about/how#retrieving-metadata)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "x509: certificate signed by unknown authority") {
				// disable TLS check, try again
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				no_tls_client := http.Client{Transport: tr}
				resp, err = no_tls_client.Do(req)
				if err != nil {
					return nil, InvalidURLError(url)
				}
				return resp, nil
			}
			continue
		} else if resp.StatusCode == http.StatusNotFound {
			continue
		} else if IsRedirect(resp.StatusCode) {
			return nil, e.ErrRedirect
		}

		return resp, nil
	}

	return nil, InvalidURLError(url)
}

func InvalidURLError(url string) error {
	return fmt.Errorf("invalid URL: %s", url)
}

func GetLinkExtraMetadataFromHTML(html_md HTMLMetadata) *model.LinkExtraMetadata {
	x_md := &model.LinkExtraMetadata{}

	switch {
	case html_md.OGDescription != "":
		x_md.AutoSummary = html_md.OGDescription
	case html_md.Description != "":
		x_md.AutoSummary = html_md.Description
	case html_md.OGTitle != "":
		x_md.AutoSummary = html_md.OGTitle
	case html_md.Title != "":
		x_md.AutoSummary = html_md.Title
	case html_md.OGSiteName != "":
		x_md.AutoSummary = html_md.OGSiteName
	}

	if html_md.OGImage != "" {
		if _, err := GetResolvedURLResponse(html_md.OGImage); err == nil {
			x_md.PreviewImgURL = html_md.OGImage
		}
	}

	return x_md
}

func IsRedirect(status_code int) bool {
	return status_code > 299 && status_code < 400
}

func SavePreviewImgAndGetFileName(url string, link_id string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("no URL provided: could not fetch preview image")
	}

	prevew_img_resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("could not fetch preview image: %s", err)
	}
	defer prevew_img_resp.Body.Close()

	img_upload := &model.ImgUpload{
		Bytes: prevew_img_resp.Body,
		Purpose: "LinkPreview",
		UID: link_id,
	}
	file_name, err := SaveUploadedImg(img_upload)
	if err != nil {
		return "", fmt.Errorf("could not save preview image: %s", err)
	}

	return file_name, nil
}

func LinkAlreadyAdded(url string) (bool, string) {
	var id sql.NullString

	err := db.Client.QueryRow("SELECT id FROM Links WHERE url = ?", url).Scan(&id)

	if err == nil && id.Valid {
		return true, id.String
	} else {
		return false, ""
	}
}

func IncrementSpellfixRanksForCats(tx *sql.Tx, cats []string) error {
	if tx != nil {
		for _, cat := range cats {

			// if word is not in global_cats_spellfix, insert it
			var rank int
			err := tx.QueryRow("SELECT rank FROM global_cats_spellfix WHERE word = ?;", cat).Scan(&rank)
			if err != nil {
				_, err = tx.Exec(
					"INSERT INTO global_cats_spellfix (word, rank) VALUES (?, ?);",
					cat,
					1,
				)
				if err != nil {
					return err
				}

				// else increment
			} else {
				_, err = tx.Exec(
					"UPDATE global_cats_spellfix SET rank = rank + 1 WHERE word = ?;",
					cat,
				)
				if err != nil {
					return err
				}
			}
			if err != nil {
				return err
			}
		}
	} else {
		for _, cat := range cats {
			var rank int
			err := db.Client.QueryRow("SELECT rank FROM global_cats_spellfix WHERE word = ?;", cat).Scan(&rank)
			if err != nil {
				_, err = db.Client.Exec(
					"INSERT INTO global_cats_spellfix (word, rank) VALUES (?, ?);",
					cat,
					1,
				)
				if err != nil {
					return err
				}
			} else {
				_, err = db.Client.Exec(
					"UPDATE global_cats_spellfix SET rank = rank + 1 WHERE word = ?;",
					cat,
				)
				if err != nil {
					return err
				}
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Delete link
func DecrementSpellfixRanksForCats(tx *sql.Tx, cats []string) error {
	if tx != nil {
		for _, cat := range cats {

			// if word has rank of 1, delete it
			var rank int
			err := tx.QueryRow("SELECT rank FROM global_cats_spellfix WHERE word = ?;", cat).Scan(&rank)
			if err != nil {
				return err
			} else if rank == 1 {
				_, err = tx.Exec(
					"DELETE FROM global_cats_spellfix WHERE word = ?;",
					cat,
				)
				if err != nil {
					return err
				}
			}
			_, err = tx.Exec(
				"UPDATE global_cats_spellfix SET rank = rank - 1 WHERE word = ?;",
				cat,
			)
			if err != nil {
				return err
			}
		}
	} else {
		for _, cat := range cats {
			_, err := db.Client.Exec(
				"UPDATE global_cats_spellfix SET rank = rank - 1 WHERE word = ?;",
				cat,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Star link
func UserSubmittedLink(login_name string, link_id string) bool {
	var sb sql.NullString
	err := db.Client.QueryRow("SELECT submitted_by FROM Links WHERE id = ?;", link_id).Scan(&sb)

	if err != nil {
		return false
	}

	return sb.String == login_name
}

func UserHasStarredLink(user_id string, link_id string) bool {
	var l sql.NullString
	err := db.Client.QueryRow(`SELECT id FROM Stars WHERE user_id = ? AND link_id = ?;`, user_id, link_id).Scan(&l)

	return err == nil && l.Valid
}

func GetUsersStarsForLink(user_id string, link_id string) uint8 {
	var stars uint8
	if err := db.Client.QueryRow(`SELECT num_stars FROM Stars WHERE user_id = ? AND link_id = ?;`, user_id, link_id).Scan(&stars); err != nil {
		log.Printf("Could not get stars for link %s: %s", link_id, err)
		return 0
	}
	return stars
}