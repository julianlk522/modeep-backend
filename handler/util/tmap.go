package handler

import (
	"github.com/julianlk522/fitm/db"

	"database/sql"
	"net/http"
	"slices"
	"strings"

	e "github.com/julianlk522/fitm/error"
	m "github.com/julianlk522/fitm/middleware"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

const TMAP_CATS_PAGE_LIMIT int = 12

// Get treasure map
func UserExists(login_name string) (bool, error) {
	var u sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Users WHERE login_name = ?;", login_name).Scan(&u)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func GetTmapForUser[T model.TmapLink | model.TmapLinkSignedIn](login_name string, r *http.Request) (interface{}, error) {

	// Queries
	// links
	submitted_sql := query.NewTmapSubmitted(login_name)
	copied_sql := query.NewTmapCopied(login_name)
	tagged_sql := query.NewTmapTagged(login_name)
	// NSFW links count
	nsfw_links_count_sql := query.NewTmapNSFWLinksCount(login_name)

	// Apply params
	// cats filter
	cats_params := r.URL.Query().Get("cats")
	has_cat_filter := cats_params != ""

	var profile *model.Profile
	if !has_cat_filter {
		var err error
		profile_sql := query.NewTmapProfile(login_name)
		profile, err = ScanTmapProfile(profile_sql)
		if err != nil {
			return nil, err
		}
	}

	// tmap queries use escaped reserved chars for MATCH clauses
	// but GetCatCountsFromTmapLinks later requires unescaped
	var cats, cats_with_unescaped_reserved_chars []string
	if has_cat_filter {
		cats = strings.Split(cats_params, ",")

		// copy before escaping reserved chars
		// need to call make() so dst is not empty and has length
		cats_with_unescaped_reserved_chars = make([]string, len(cats))
		copy(cats_with_unescaped_reserved_chars, cats)

		query.EscapeCatsReservedChars(cats)

		submitted_sql = submitted_sql.FromCats(cats)
		copied_sql = copied_sql.FromCats(cats)
		tagged_sql = tagged_sql.FromCats(cats)
		nsfw_links_count_sql = nsfw_links_count_sql.FromCats(cats)
	}

	// auth (add IsLiked, IsCopied)
	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["user_id"].(string)
	if req_user_id != "" {
		submitted_sql = submitted_sql.AsSignedInUser(req_user_id)
		copied_sql = copied_sql.AsSignedInUser(req_user_id)
		tagged_sql = tagged_sql.AsSignedInUser(req_user_id)
	}

	// nsfw
	var nsfw_params string
	if r.URL.Query().Get("nsfw") != "" {
		nsfw_params = r.URL.Query().Get("nsfw")
	} else if r.URL.Query().Get("NSFW") != "" {
		nsfw_params = r.URL.Query().Get("NSFW")
	}

	if nsfw_params == "true" {
		submitted_sql = submitted_sql.NSFW()
		copied_sql = copied_sql.NSFW()
		tagged_sql = tagged_sql.NSFW()
	} else if nsfw_params != "false" && nsfw_params != "" {
		return nil, e.ErrInvalidNSFWParams
	}

	// Scan
	// links
	submitted, err := ScanTmapLinks[T](submitted_sql.Query)
	if err != nil {
		return nil, err
	}
	copied, err := ScanTmapLinks[T](copied_sql.Query)
	if err != nil {
		return nil, err
	}
	tagged, err := ScanTmapLinks[T](tagged_sql.Query)
	if err != nil {
		return nil, err
	}
	// NSFW links count
	var nsfw_links_count int
	if err := db.Client.QueryRow(nsfw_links_count_sql.Text, nsfw_links_count_sql.Args...).Scan(&nsfw_links_count); err != nil {
		return nil, err
	}

	// Get cat counts from links
	all_links := slices.Concat(*submitted, *copied, *tagged)
	var cat_counts *[]model.CatCount
	if has_cat_filter {
		cat_counts = GetCatCountsFromTmapLinks(
			&all_links,
			&model.TmapCatCountsOpts{
				OmittedCats: cats_with_unescaped_reserved_chars,
			},
		)
	} else {
		cat_counts = GetCatCountsFromTmapLinks(&all_links, nil)
	}

	// Assemble and return tmap
	sections := &model.TmapSections[T]{
		Cats:      cat_counts,
		Submitted: submitted,
		Copied:    copied,
		Tagged:    tagged,
	}

	if has_cat_filter {
		return model.FilteredTmap[T]{
			TmapSections: sections,
			NSFWLinksCount: nsfw_links_count,
		}, nil

	} else {
		return model.Tmap[T]{
			Profile:      profile,
			TmapSections: sections,
			NSFWLinksCount: nsfw_links_count,
		}, nil
	}
}

func ScanTmapProfile(sql *query.TmapProfile) (*model.Profile, error) {
	var u model.Profile
	err := db.Client.
		QueryRow(sql.Text, sql.Args...).
		Scan(
			&u.LoginName,
			&u.About,
			&u.PFP,
			&u.Created,
		)
	if err != nil {
		return nil, e.ErrNoUserWithLoginName
	}

	return &u, nil
}

func ScanTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](sql *query.Query) (*[]T, error) {
	rows, err := db.Client.Query(sql.Text, sql.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links interface{}

	switch any(new(T)).(type) {
	case *model.TmapLinkSignedIn:
		var signed_in_links = []model.TmapLinkSignedIn{}

		for rows.Next() {
			l := model.TmapLinkSignedIn{}
			err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.CatsFromUser,
				&l.Summary,
				&l.SummaryCount,
				&l.LikeCount,
				&l.TagCount,
				&l.ImgURL,

				// Add IsLiked / IsCopied
				&l.IsLiked,
				&l.IsCopied)
			if err != nil {
				return nil, err
			}
			signed_in_links = append(signed_in_links, l)
		}

		links = &signed_in_links

	case *model.TmapLink:
		var signed_out_links = []model.TmapLink{}

		for rows.Next() {
			l := model.TmapLink{}
			err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.CatsFromUser,
				&l.Summary,
				&l.SummaryCount,
				&l.LikeCount,
				&l.TagCount,
				&l.ImgURL)
			if err != nil {
				return nil, err
			}
			signed_out_links = append(signed_out_links, l)
		}

		links = &signed_out_links
	}

	return links.(*[]T), nil
}

// Get counts of each category found in links
// Omit any cats passed via opts.OmittedCats
// (omit used to retrieve subcats by passing directly searched cats)
// TODO: refactor to make this clearer?
func GetCatCountsFromTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](links *[]T, opts *model.TmapCatCountsOpts) *[]model.CatCount {
	counts := []model.CatCount{}
	found_cats := []string{}
	var found bool

	for _, link := range *links {
		var cats string
		switch l := any(link).(type) {
		case model.TmapLinkSignedIn:
			cats = l.Cats
		case model.TmapLink:
			cats = l.Cats
		}

		for _, cat := range strings.Split(cats, ",") {
			if opts != nil &&
				slices.Contains(opts.OmittedCats, cat) {
				continue
			}

			found = false
			for _, found_cat := range found_cats {
				if cat == found_cat {
					found = true

					for i, count := range counts {
						if count.Category == cat {
							counts[i].Count++
							break
						}
					}
				}
			}

			if !found {
				counts = append(counts, model.CatCount{Category: cat, Count: 1})
				found_cats = append(found_cats, cat)
			}
		}
	}

	SortAndLimitCatCounts(&counts, TMAP_CATS_PAGE_LIMIT)

	return &counts
}

func SortAndLimitCatCounts(cat_counts *[]model.CatCount, limit int) {
	slices.SortFunc(*cat_counts, model.SortCats)

	if len(*cat_counts) > limit {
		*cat_counts = (*cat_counts)[:limit]
	}
}
