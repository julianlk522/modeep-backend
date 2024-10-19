package handler

import (
	"strings"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"

	"database/sql"
	"fmt"

	"net/http"

	"github.com/go-chi/render"
)

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

func RenderZeroLinks(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &model.PaginatedLinks[model.Link]{NextPage: -1})
	render.Status(r, http.StatusOK)
}

func ScanLinks[T model.Link | model.LinkSignedIn](get_links_sql *query.TopLinks) (*[]T, error) {
	rows, err := db.Client.Query(get_links_sql.Text, get_links_sql.Args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var links interface{}

	switch any(new(T)).(type) {
	case *model.Link:
		var signed_out_links = []model.Link{}

		for rows.Next() {
			i := model.Link{}
			err := rows.Scan(
				&i.ID,
				&i.URL,
				&i.SubmittedBy,
				&i.SubmitDate,
				&i.Cats,
				&i.Summary,
				&i.SummaryCount,
				&i.TagCount,
				&i.LikeCount,
				&i.ImgURL,
			)
			if err != nil {
				return nil, err
			}
			signed_out_links = append(signed_out_links, i)
		}

		links = &signed_out_links

	case *model.LinkSignedIn:
		var signed_in_links = []model.LinkSignedIn{}

		for rows.Next() {
			i := model.LinkSignedIn{}
			if err := rows.Scan(
				&i.ID,
				&i.URL,
				&i.SubmittedBy,
				&i.SubmitDate,
				&i.Cats,
				&i.Summary,
				&i.SummaryCount,
				&i.TagCount,
				&i.LikeCount,
				&i.ImgURL,
				&i.IsLiked,
				&i.IsCopied,
			); err != nil {
				return nil, err
			}

			signed_in_links = append(signed_in_links, i)
		}

		links = &signed_in_links
	}

	return links.(*[]T), nil
}

func PaginateLinks[T model.LinkSignedIn | model.Link](links *[]T, page int) interface{} {
	if links == nil || len(*links) == 0 {
		return &model.PaginatedLinks[model.Link]{NextPage: -1}
	}

	if len(*links) == query.LINKS_PAGE_LIMIT+1 {
		sliced := (*links)[0:query.LINKS_PAGE_LIMIT]
		return &model.PaginatedLinks[T]{
			NextPage: page + 1,
			Links:    &sliced,
		}
	} else {
		return &model.PaginatedLinks[T]{
			NextPage: -1,
			Links:    links,
		}
	}
}

// Add link (non-YT)
func ObtainURLMetaData(request *model.NewLinkRequest) error {
	resp, err := GetResolvedURLResponse(request.NewLink.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// save updated URL (after any redirects e.g., to wwww.)
	request.URL = resp.Request.URL.String()

	// remove trailing slash
	request.URL = strings.TrimSuffix(request.URL, "/")

	if resp.StatusCode != http.StatusForbidden {
		meta := ExtractMetaFromHTMLTokens(resp.Body)
		AssignMetadata(meta, request)
	}

	return nil
}
func GetResolvedURLResponse(url string) (*http.Response, error) {
	protocols := []string{"", "https://", "http://"}

	for _, prefix := range protocols {
		full_url := prefix + url

		// set FITM user agent
		req, err := http.NewRequest("GET", full_url, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "FITM-Bot (https://fitm.online/about#retrieving-metadata)")
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode == http.StatusNotFound {
			continue
		} else if IsRedirect(resp.StatusCode) {
			return nil, e.ErrRedirect
		}

		// URL is valid: return
		return resp, nil
	}

	return nil, InvalidURLError(url)
}

func InvalidURLError(url string) error {
	return fmt.Errorf("invalid URL: %s", url)
}

func AssignMetadata(meta HTMLMeta, link_data *model.NewLinkRequest) {
	switch {
	case meta.OGDescription != "":
		link_data.AutoSummary = meta.OGDescription
	case meta.Description != "":
		link_data.AutoSummary = meta.Description
	case meta.OGTitle != "":
		link_data.AutoSummary = meta.OGTitle
	case meta.Title != "":
		link_data.AutoSummary = meta.Title
	case meta.OGSiteName != "":
		link_data.AutoSummary = meta.OGSiteName
	}

	if meta.OGImage != "" {
		resp, err := http.Get(meta.OGImage)
		if err == nil && resp.StatusCode != 404 && !IsRedirect(resp.StatusCode) {
			link_data.ImgURL = meta.OGImage
		}
	}
}

func IsRedirect(status_code int) bool {
	return status_code > 299 && status_code < 400
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

	// exec as part of transaction if tx is not nil
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

		// else run against DB connection directly
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

	// exec as part of transaction if tx is not nil
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

		// else run against DB connection directly
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
