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
	"net/url"
)

var Preview_img_dir string

func init() {
	backend_root_path := os.Getenv("MODEEP_BACKEND_ROOT")
	if backend_root_path == "" {
		log.Panic("$MODEEP_BACKEND_ROOT not set")
	}
	Preview_img_dir = backend_root_path + "/db/img/preview"
}

func GetTopLinksOptionsFromRequestParams(params url.Values) (*model.TopLinksOptions, error) {
	opts := &model.TopLinksOptions{}

	// For cats that the links must have
	cats_params := params.Get("cats")
	if cats_params != "" {
		opts.CatFiltersWithSpellingVariants = query.GetCatsOptionalPluralOrSingularForms(
			strings.Split(cats_params, ","),
		)
	}
	// For cats that the links must NOT have
	neutered_params := params.Get("neutered")
	if neutered_params != "" {
		// Since we use IN, not FTS MATCH, spelling variants are not
		// needed (and casing matters)
		opts.NeuteredCatFilters = strings.Split(neutered_params, ",")
	}
	summary_contains_params := params.Get("summary_contains")
	if summary_contains_params != "" {
		opts.GlobalSummaryContains = summary_contains_params
	}
	url_contains_params := params.Get("url_contains")
	if url_contains_params != "" {
		opts.URLContains = url_contains_params
	}
	url_lacks_params := params.Get("url_lacks")
	if url_lacks_params != "" {
		opts.URLLacks = url_lacks_params
	}
	var nsfw_params string
	if params.Get("include_nsfw") != "" {
		nsfw_params = params.Get("include_nsfw")
	}
	if nsfw_params == "true" {
		opts.IncludeNSFW = true
	} else if nsfw_params != "false" && nsfw_params != "" {
		return nil, e.ErrInvalidNSFWParams
	}
	var sort_by model.SortBy = model.SortByTimesStarred
	sort_params := params.Get("sort_by")
	if sort_params != "" {
		sort_by = model.SortBy(sort_params)
		found := false
		for _, sb := range model.ValidSortBys {
			if sb == sort_by {
				opts.SortBy = sort_by
				found = true
				break
			}
		}
		if !found {
			return nil, e.ErrInvalidSortByParams
		}
	}
	period_params := params.Get("period")
	if period_params != "" {
		period := model.Period(period_params)
		if _, ok := model.ValidPeriodsInDays[period]; !ok {
			return nil, e.ErrInvalidPeriod
		}
		opts.Period = period
	}
	// AsSignedInUser and Page are added to opts directly in GetTopLinks() handler after middleware processing
	return opts, nil
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

	return count >= MAX_DAILY_SUBMITTED_LINKS, nil
}

func PrepareLinksPage[T model.HasCats](links_sql *query.TopLinks, options *model.LinksPageOptions) (*model.LinksPage[T], error) {
	links_page, err := scanRawLinksPageData[T](links_sql)
	if err != nil {
		return nil, err
	}

	paginateLinks(links_page.Links)

	cat_filters := options.CatFilters
	if len(cat_filters) > 0 {
		links_page.MergedCats = getMergedCatSpellingVariantsInLinksFromCatFilters(
			links_page.Links,
			cat_filters,
		)
	}

	hidden_links, err := getNSFWLinksCount[T](links_sql)
	if err != nil {
		return nil, err
	}

	links_page.NSFWLinksCount = hidden_links

	return links_page, nil
}

func scanRawLinksPageData[T model.Link | model.LinkSignedIn](links_sql *query.TopLinks) (*model.LinksPage[T], error) {
	if links_sql.Error != nil {
		return nil, links_sql.Error
	}

	rows, err := links_sql.ValidateAndExecuteRows()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	// NOTE: this both scans the links and creates the LinksPage struct
	// because number of pages is taken here from the query result rows
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
				&l.AvgStars,
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
				&l.AvgStars,
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
	row, err := single_link_sql.ValidateAndExecuteRow()
	if err != nil {
		return nil, err
	}

	var link any
	switch any(new(T)).(type) {
	case *model.LinkSignedIn:
		var l = &model.LinkSignedIn{}
		if err := row.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
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
		if err := row.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
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

func paginateLinks[T model.LinkSignedIn | model.Link](links *[]T) {
	if links == nil || len(*links) == 0 {
		return
	} else if len(*links) == query.LINKS_PAGE_LIMIT+1 {
		*links = (*links)[0:query.LINKS_PAGE_LIMIT]
	}
}

func getMergedCatSpellingVariantsInLinksFromCatFilters[T model.HasCats](links *[]T, cat_filters []string) []string {
	if links == nil || len(*links) == 0 {
		return nil
	}

	var merged_cats []string
	for _, link := range *links {
		link_cats := strings.SplitSeq(link.GetCats(), ",")
		for cat := range link_cats {
			cat_lc := strings.ToLower(cat)

			// Merge if does not match cat filter exactly but is close
			if !slices.Contains(cat_filters, cat) {
				for _, cf := range cat_filters {
					cf_lc := strings.ToLower(cf)

					if CatsResembleEachOther(cat_lc, cf_lc) &&
						!slices.Contains(merged_cats, cat) {
						merged_cats = append(merged_cats, cat)
					}
				}
			}
		}
	}

	return merged_cats
}

func getNSFWLinksCount[T model.HasCats](links_sql *query.TopLinks) (int, error) {
	hidden_links_count_sql := links_sql.CountNSFWLinks()
	row, err := hidden_links_count_sql.ValidateAndExecuteRow()
	if err != nil {
		return 0, err
	}

	var hidden_links sql.NullInt32
	if err := row.Scan(&hidden_links); err != nil {
		return 0, err
	}

	return int(hidden_links.Int32), nil
}

// Add link (non-YT)
func GetLinkExtraMetadataFromResponse(resp *http.Response) *model.LinkExtraMetadata {
	if resp == nil {
		return nil
	} else if resp.StatusCode != http.StatusForbidden {
		html_md := extractHTMLMetadata(resp.Body)
		return getLinkExtraMetadataFromHTML(resp.Request.URL, html_md)
	}

	return nil
}

func getLinkExtraMetadataFromHTML(url *url.URL, html_md HTMLMetadata) *model.LinkExtraMetadata {
	x_md := &model.LinkExtraMetadata{}

	switch {
	case html_md.OGDesc != "":
		x_md.AutoSummary = html_md.OGDesc
	case html_md.Desc != "":
		x_md.AutoSummary = html_md.Desc
	case html_md.OGTitle != "":
		x_md.AutoSummary = html_md.OGTitle
	case html_md.Title != "":
		x_md.AutoSummary = html_md.Title
	case html_md.OGSiteName != "":
		x_md.AutoSummary = html_md.OGSiteName
	case html_md.TwitterDesc != "":
		x_md.AutoSummary = html_md.TwitterDesc
	case html_md.TwitterTitle != "":
		x_md.AutoSummary = html_md.TwitterTitle
	}

	// Test preview image URL to confirm it can be accessed
	// TODO cleanup
	if html_md.OGImage != "" {
		if !strings.HasPrefix(html_md.OGImage, "http") {
			html_md.OGImage = url.Scheme + "://" + url.Host + "/" + html_md.OGImage
			log.Printf("updated preview img URL: %v\n", html_md.OGImage)
		}
		if _, err := GetResolvedURLResponse(html_md.OGImage); err == nil {
			x_md.PreviewImgURL = html_md.OGImage
		}
	} else if html_md.TwitterImage != "" {
		if !strings.HasPrefix(html_md.TwitterImage, "http") {
			html_md.TwitterImage = url.Scheme + "://" + url.Host + "/" + html_md.TwitterImage
			log.Printf("updated preview img URL: %v\n", html_md.OGImage)
		}
		if _, err := GetResolvedURLResponse(html_md.TwitterImage); err == nil {
			x_md.PreviewImgURL = html_md.TwitterImage
		}
	}

	return x_md
}

func GetResolvedURLResponse(url string) (*http.Response, error) {
	var urls_to_try []string
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		urls_to_try = []string{url}
	} else {
		urls_to_try = []string{"https://" + url, "http://" + url}
	}
	for _, full_url := range urls_to_try {
		req, err := http.NewRequest("GET", full_url, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", MODEEP_BOT_USER_AGENT)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		// Skip "Accept-Encoding" to keep default Go HTTP client decompression
		// (Restoring decompression behavior is annoying...
		// Can reconsider this if omitting Accept-Encoding becomes a problem.)
		// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// disable TLS check, try again
			if strings.Contains(err.Error(), "x509: certificate signed by unknown authority") {
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				no_tls_client := http.Client{Transport: tr}
				resp, err = no_tls_client.Do(req)
				if err != nil {
					return nil, invalidURLError(url)
				}
			} else {
				continue
			}
		}

		if resp == nil {
			continue
		}

		if resp.StatusCode == http.StatusBadRequest {
			// Other 400+ status codes are OK - e.g., 403 maybe means the site is real and they just don't like the
			// Modeep-Bot user agent. Or a 500 could mean down temporarily. Can give the benefit of the doubt in those
			// cases.
			continue
		} else if isRedirect(resp.StatusCode) {
			return nil, e.ErrRedirect
		}

		return resp, nil
	}

	return nil, invalidURLError(url)
}

func invalidURLError(url string) error {
	return fmt.Errorf("invalid URL: %s", url)
}

func isRedirect(status_code int) bool {
	return status_code >= http.StatusMultipleChoices &&
		status_code < http.StatusBadRequest
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
		Bytes:   prevew_img_resp.Body,
		Purpose: "LinkPreview",
		UID:     link_id,
	}
	file_name, err := SaveUploadedImgAndGetNewFileName(img_upload)
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
	cats = getDeduplicatedCats(cats)
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
	cats = getDeduplicatedCats(cats)
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
			// else decrement
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

// "ham,Ham,cheese,cHeEsE" -> "ham,cheese"
func getDeduplicatedCats(cats []string) []string {
	seen := make(map[string]string)
	deduped_cats := make([]string, 0, len(cats))

	for _, cat := range cats {
		lower := strings.ToLower(cat)
		if _, exists := seen[lower]; !exists {
			seen[lower] = cat
			deduped_cats = append(deduped_cats, cat)
		}
	}

	return deduped_cats
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
