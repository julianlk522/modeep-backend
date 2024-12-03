package handler

import (
	"math"
	"strconv"

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
	var links_options = &model.TmapLinksOptions{}
	nsfw_links_count_sql := query.NewTmapNSFWLinksCount(login_name)

	cats_params := r.URL.Query().Get("cats")
	has_cat_filter := cats_params != ""

	// add profile only if unfiltered
	var profile *model.Profile
	if !has_cat_filter {
		var err error
		profile_sql := query.NewTmapProfile(login_name)
		profile, err = ScanTmapProfile(profile_sql)
		if err != nil {
			return nil, err
		}
	} else {
		cats := strings.Split(cats_params, ",")
		// tmap queries use escaped reserved chars and both singular/plural
		// forms for MATCH clauses
		query.EscapeCatsReservedChars(cats)
		cats = query.GetCatsOptionalPluralOrSingularForms(cats)
		nsfw_links_count_sql = nsfw_links_count_sql.FromCats(cats)
		links_options.CatsFilter = cats
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["user_id"].(string)
	if req_user_id != "" {
		links_options.AsSignedInUser = req_user_id
	}

	var nsfw_params string
	if r.URL.Query().Get("nsfw") != "" {
		nsfw_params = r.URL.Query().Get("nsfw")
	} else if r.URL.Query().Get("NSFW") != "" {
		nsfw_params = r.URL.Query().Get("NSFW")
	}

	if nsfw_params == "true" {
		links_options.IncludeNSFW = true
	} else if nsfw_params != "false" && nsfw_params != "" {
		return nil, e.ErrInvalidNSFWParams
	}

	sort_params := r.URL.Query().Get("sort_by")
	if sort_params == "newest" {
		links_options.SortByNewest = true
	} else if sort_params != "rating" && sort_params != "" {
		return nil, e.ErrInvalidSortByParams
	}
	
	var nsfw_links_count int

	section := strings.ToLower(r.URL.Query().Get("section"))
	// single section
	if section != "" {
		var links *[]T
		var err error

		switch section {
		case "submitted":
			links, err = ScanTmapLinks[T](query.NewTmapSubmitted(login_name).FromOptions(links_options).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.SubmittedOnly()
		case "copied":
			links, err = ScanTmapLinks[T](query.NewTmapCopied(login_name).FromOptions(links_options).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.CopiedOnly()
		case "tagged":
			links, err = ScanTmapLinks[T](query.NewTmapTagged(login_name).FromOptions(links_options).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.TaggedOnly()
		default:
			return nil, e.ErrInvalidSection
		}
		if err != nil {
			return nil, err
		}

		if len(*links) == 0 {
			return model.PaginatedTmapSection[T]{
				Links: &[]T{},
				Cats:  &[]model.CatCount{},
				NSFWLinksCount: 0,
				NextPage: -1,
			}, nil
		}

		var cat_counts *[]model.CatCount
		if has_cat_filter {
			// cats_to_not_count have unescaped reserved chars
			// they are lowercased to check against all capitalization variants
			cats_to_not_count := strings.Split(strings.ToLower(cats_params), ",")
			cat_counts = GetCatCountsFromTmapLinks(
				links,
				&model.TmapCatCountsOpts{
					OmittedCats: cats_to_not_count,
				},
			)
		} else {
			cat_counts = GetCatCountsFromTmapLinks(links, nil)
		}

		// Paginate section links
		// due to counting cats manually (i.e., not in SQL) the pagination
		// must also be done manually after retrieving the full slice of links
		// and counting cats
		page, next_page := 1, -1
		page_params := r.URL.Query().Get("page")
		if page_params != "" && page_params != "0" {
			page, err = strconv.Atoi(page_params)
			if err != nil {
				return nil, e.ErrInvalidPage
			}
		}
		total_pages := int(math.Ceil(float64(len(*links)) / float64(query.LINKS_PAGE_LIMIT)))
		if page > total_pages {
			links = &[]T{}
		} else if page == total_pages {
			*links = (*links)[query.LINKS_PAGE_LIMIT*(page-1):]
		} else {
			*links = (*links)[query.LINKS_PAGE_LIMIT*(page-1):query.LINKS_PAGE_LIMIT*page]
			next_page = page + 1
		}

		if err := db.Client.QueryRow(nsfw_links_count_sql.Text, nsfw_links_count_sql.Args...).Scan(&nsfw_links_count); err != nil {
			return nil, err
		}
		
		return model.PaginatedTmapSection[T]{
			Links: links,
			Cats:  cat_counts,
			NextPage: next_page,
			NSFWLinksCount: nsfw_links_count,
		}, nil

	// all sections
	} else {
		
		// 20+ links: indicate in response so can be paginated
		var sections_with_more []string

		submitted, err := ScanTmapLinks[T](query.NewTmapSubmitted(login_name).FromOptions(links_options).Query)
		if err != nil {
			return nil, err
		}
		if len(*submitted) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "submitted")
			*submitted = (*submitted)[0:query.LINKS_PAGE_LIMIT]
		}
		copied, err := ScanTmapLinks[T](query.NewTmapCopied(login_name).FromOptions(links_options).Query)
		if err != nil {
			return nil, err
		}
		if len(*copied) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "copied")
			*copied = (*copied)[0:query.LINKS_PAGE_LIMIT]
		}
		tagged, err := ScanTmapLinks[T](query.NewTmapTagged(login_name).FromOptions(links_options).Query)
		if err != nil {
			return nil, err
		}
		if len(*tagged) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "tagged")
			*tagged = (*tagged)[0:query.LINKS_PAGE_LIMIT]
		}

		all_links := slices.Concat(*submitted, *copied, *tagged)
		if len(all_links) == 0 {
			return model.FilteredTmap[T]{
				TmapSections:   nil,
				NSFWLinksCount: 0,
			}, nil
		}

		var cat_counts *[]model.CatCount
		if has_cat_filter {
			// cats_to_not_count have unescaped reserved chars
			// they are lowercased to check against all capitalization variants
			cats_to_not_count := strings.Split(strings.ToLower(cats_params), ",")
			cat_counts = GetCatCountsFromTmapLinks(
				&all_links,
				&model.TmapCatCountsOpts{
					OmittedCats: cats_to_not_count,
				},
			)
		} else {
			cat_counts = GetCatCountsFromTmapLinks(&all_links, nil)
		}

		sections := &model.TmapSections[T]{
			Submitted: submitted,
			Copied:    copied,
			Tagged:    tagged,
			SectionsWithMore: sections_with_more,
			Cats:      cat_counts,
		}

		if err := db.Client.QueryRow(nsfw_links_count_sql.Text, nsfw_links_count_sql.Args...).Scan(&nsfw_links_count); err != nil {
			return nil, err
		}

		if has_cat_filter {
			return model.FilteredTmap[T]{
				TmapSections:   sections,
				NSFWLinksCount: nsfw_links_count,
			}, nil

		} else {
			return model.Tmap[T]{
				Profile:        profile,
				TmapSections:   sections,
				NSFWLinksCount: nsfw_links_count,
			}, nil
		}
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

				// signed-in only properties
				&l.IsLiked,
				&l.IsCopied)
			if err != nil {
				return nil, err
			}
			signed_in_links = append(signed_in_links, l)
		}

		links = &signed_in_links
	}

	return links.(*[]T), nil
}

func GetCatCountsFromTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](links *[]T, opts *model.TmapCatCountsOpts) *[]model.CatCount {
	counts := []model.CatCount{}
	found_cats := []string{}
	var found bool

	for _, link := range *links {
		var cats string
		switch l := any(link).(type) {
		case model.TmapLink:
			cats = l.Cats
		case model.TmapLinkSignedIn:
			cats = l.Cats
		}

		for _, cat := range strings.Split(cats, ",") {
			if strings.TrimSpace(cat) == "" || (opts != nil &&
				slices.Contains(opts.OmittedCats, strings.ToLower(cat))) {
				continue
			}

			found = false
			for _, found_cat := range found_cats {
				if found_cat == cat {
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

	slices.SortFunc(counts, model.SortCats)

	if opts != nil {
		MergeCatCountsCapitalizationVariants(&counts,opts.OmittedCats)
	}

	if len(counts) > TMAP_CATS_PAGE_LIMIT {
		counts = (counts)[:TMAP_CATS_PAGE_LIMIT]
	}
	
	return &counts
}

// merge counts of capitalization variants e.g. "Music" and "music"
func MergeCatCountsCapitalizationVariants(counts *[]model.CatCount, omitted_cats []string) {
	for i, count := range *counts {
		for j := i + 1; j < len(*counts); j++ {
			if strings.EqualFold(count.Category, (*counts)[j].Category) {

				// skip if is some capitalization variant of a cat filter
				if len(omitted_cats) > 0 && slices.Contains(omitted_cats, strings.ToLower((*counts)[j].Category)) {
					continue
				}
				(*counts)[i].Count += (*counts)[j].Count
				*counts = append((*counts)[:j], (*counts)[j+1:]...)
			}
		}
	}
}