package handler

import (
	"crypto/tls"
	"io"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"

	"database/sql"
	"fmt"

	"net/http"
)

// capitalized so it can be exported and used in GetPreviewImg handler
var Preview_img_dir string

func init() {
	test_data_path := os.Getenv("FITM_TEST_DATA_PATH")
	if test_data_path == "" {
		log.Panic("FITM_TEST_DATA_PATH not set")
	}
	Preview_img_dir = test_data_path + "/img/preview"
}

const MAX_DAILY_LINKS = 50

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

func PrepareLinksResponse[T model.HasCats](links_sql *query.TopLinks, cats_params string) (*model.LinksPage[T], error) {
	links_page, err := ScanLinks[T](links_sql)
	if err != nil {
		return nil, err
	}
	
	PaginateLinks(links_page.Links)
	if cats_params != "" {
		CountMergedCatSpellingVariants(links_page, cats_params)
	}

	return links_page, nil
}

func ScanLinks[T model.Link | model.LinkSignedIn](links_sql *query.TopLinks) (*model.LinksPage[T], error) {
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
	// because page_count must be taken here from the query result rows
	var links any
	var page_count int

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
				&l.LikeCount,
				&l.CopyCount,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,
				&page_count,
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
				&l.LikeCount,
				&l.CopyCount,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,
				&page_count,
				&l.IsLiked,
				&l.IsCopied,
			); err != nil {
				return nil, err
			}

			signed_in_links = append(signed_in_links, l)
		}

		links = &signed_in_links
	}

	if links == nil || len(*links.(*[]T)) == 0 {
		return &model.LinksPage[T]{PageCount: -1}, nil
	}

	return &model.LinksPage[T]{
		Links: links.(*[]T),
		PageCount: page_count,
	}, nil
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
		link_cats := strings.Split(strings.ToLower(link.GetCats()), ",")
		has_cat_from_filters := false
		for _, cat := range link_cats {
			if slices.Contains(cat_filters, cat) {
				has_cat_from_filters = true
				break
			}
		}

		if !has_cat_from_filters {
			// find out which cat(s) spelling variants were added
			// and add them to MergedCats so that frontend can alert user
			for _, lc := range link_cats {
				for _, cf := range cat_filters {
					if CatsAreSingularOrPluralVariationsOfEachOther(lc, cf) && !slices.Contains(lp.MergedCats, lc) {
						lp.MergedCats = append(lp.MergedCats, lc)
					}
				}
			}

		}
	}
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
		req.Header.Set("User-Agent", "FITM-Bot (https://fitm.online/about/how#retrieving-metadata)")

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
		resp, err := http.Get(html_md.OGImage)
		if err == nil && resp.StatusCode != 404 && !IsRedirect(resp.StatusCode) {
			x_md.PreviewImgURL = html_md.OGImage
		}
	}

	return x_md
}

func IsRedirect(status_code int) bool {
	return status_code > 299 && status_code < 400
}

func SavePreviewImgAndGetFileName(url string, link_id string) string {
	if url == "" {
		log.Printf("no URL provided: could not fetch preview image")
		return ""
	}

	prevew_img_resp, err := http.Get(url)
	if err != nil {
		log.Printf("could not fetch preview image: %s", err)
		return ""
	}
	defer prevew_img_resp.Body.Close()

	var extension string
	if strings.Contains(url, ".jpg") {
		extension = ".jpg"
	} else if strings.Contains(url, ".png") {
		extension = ".png"
	} else if strings.Contains(url, ".jpeg") {
		extension = ".jpeg"
	} else if strings.Contains(url, ".webp") {
		extension = ".webp"
	}

	pic_file_name := link_id + extension
	pic_file, err := os.Create(Preview_img_dir + "/" + pic_file_name)
	if err != nil {
		log.Printf("could not create new preview image: %s", err)
	}
	defer pic_file.Close()

	if _, err = io.Copy(pic_file, prevew_img_resp.Body); err != nil {
		log.Printf("could not copy preview image: %s", err)
	}

	return pic_file_name
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

// Like / unlike link
func UserSubmittedLink(login_name string, link_id string) bool {
	var sb sql.NullString
	err := db.Client.QueryRow("SELECT submitted_by FROM Links WHERE id = ?;", link_id).Scan(&sb)

	if err != nil {
		return false
	}

	return sb.String == login_name
}

func UserHasLikedLink(user_id string, link_id string) bool {
	var l sql.NullString
	err := db.Client.QueryRow(`SELECT id FROM "Link Likes" WHERE user_id = ? AND link_id = ?;`, user_id, link_id).Scan(&l)

	return err == nil && l.Valid
}

// Copy link
func UserHasCopiedLink(user_id string, link_id string) bool {
	var l sql.NullString
	err := db.Client.QueryRow(`SELECT id 
		FROM "Link Copies" 
		WHERE user_id = ? 
		AND link_id = ?;`,
		user_id,
		link_id,
	).Scan(&l)

	return err == nil && l.Valid
}
