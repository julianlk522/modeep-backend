package handler

import (
	"database/sql"
	"math"
	"net/url"
	"strconv"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/julianlk522/modeep/db"

	"slices"
	"strings"

	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func GetTmapOptsFromRequestParams(params url.Values) (*model.TmapOptions, error) {
	var opts = &model.TmapOptions{}
	var cat_filters_params,
		period_params,
		summary_contains_params,
		url_contains_params,
		url_lacks_params,
		nsfw_params,
		sort_params,
		section_params,
		page_params string

	cat_filters_params = params.Get("cats")
	if cat_filters_params != "" {
		opts.RawCatFiltersParams = cat_filters_params
		cat_filters_with_spelling_variants := query.GetCatsOptionalPluralOrSingularForms(
			strings.Split(cat_filters_params, ","),
		)
		opts.CatFiltersWithSpellingVariants = cat_filters_with_spelling_variants
	}

	period_params = params.Get("period")
	if period_params != "" {
		opts.Period = period_params
	}

	summary_contains_params = params.Get("summary_contains")
	if summary_contains_params != "" {
		opts.SummaryContains = summary_contains_params
	}

	url_contains_params = params.Get("url_contains")
	if url_contains_params != "" {
		opts.URLContains = url_contains_params
	}

	url_lacks_params = params.Get("url_lacks")
	if url_lacks_params != "" {
		opts.URLLacks = url_lacks_params
	}

	if params.Get("nsfw") != "" {
		nsfw_params = params.Get("nsfw")
	} else if params.Get("NSFW") != "" {
		nsfw_params = params.Get("NSFW")
	}
	if nsfw_params == "true" {
		opts.IncludeNSFW = true
	} else if nsfw_params != "false" && nsfw_params != "" {
		return nil, e.ErrInvalidNSFWParams
	}

	sort_params = params.Get("sort_by")
	if sort_params != "" && sort_params != "times_starred" {
		opts.SortBy = sort_params
	}

	section_params = strings.ToLower(params.Get("section"))
	if section_params != "" {
		switch section_params {
		case "submitted", "starred", "tagged":
			opts.Section = section_params
		default:
			return nil, e.ErrInvalidSectionParams
		}
	}

	page_params = params.Get("page")
	if page_params != "" && page_params != "0" {
		page, err := strconv.Atoi(page_params)
		if err != nil || page < 1 {
			return nil, e.ErrInvalidPageParams
		}
		opts.Page = page
	}

	return opts, nil
}

func BuildTmapFromOpts[T model.TmapLink | model.TmapLinkSignedIn](opts *model.TmapOptions) (any, error) {
	if opts.OwnerLoginName == "" {
		return nil, e.ErrNoTmapOwnerLoginName
	}
	tmap_owner := opts.OwnerLoginName

	// Get number of NSFW links (so can indicate that they are hidden)
	var nsfw_links_count int
	nsfw_links_count_opts := &model.TmapNSFWLinksCountOptions{
		OnlySection:                    opts.Section,
		CatFiltersWithSpellingVariants: opts.CatFiltersWithSpellingVariants,
		Period:                         opts.Period,
		SummaryContains:                opts.SummaryContains,
		URLContains:                    opts.URLContains,
		URLLacks:                       opts.URLLacks,
	}
	nsfw_links_count_sql := query.
		NewTmapNSFWLinksCount(tmap_owner).
		FromOptions(nsfw_links_count_opts)
	row, err := nsfw_links_count_sql.ValidateAndExecuteRow()
	if err != nil {
		return nil, err
	}
	if err := row.Scan(&nsfw_links_count); err != nil {
		return nil, err
	}

	// Cat filters
	var cat_counts_opts *model.TmapCatCountsOptions
	var cat_filters []string
	has_cat_filter := len(opts.CatFiltersWithSpellingVariants) > 0
	if has_cat_filter {
		cat_counts_opts = &model.TmapCatCountsOptions{
			RawCatsParams: opts.RawCatFiltersParams,
		}

		cat_filters = strings.Split(opts.RawCatFiltersParams, ",")
	}

	// Individual section
	if opts.Section != "" {
		section_query_builder, err := getTmapLinksQueryBuilderForSectionForOwner(opts.Section, tmap_owner)
		if err != nil {
			return nil, err
		}

		links, err := buildTmapLinksQueryAndScan[T](section_query_builder, opts)
		if links == nil || len(*links) == 0 {
			return model.TmapIndividualSectionPage[T]{
				Links:          &[]T{},
				Cats:           &[]model.CatCount{},
				NSFWLinksCount: 0,
				Pages:          -1,
			}, nil
		}

		// Get cat counts
		cat_counts := getCatCountsFromTmapLinks(links, cat_counts_opts)

		// Pagination
		links, pages, err := getPageOfIndividualTmapSectionLinks(opts.Page, links)
		if err != nil {
			return nil, err
		}

		if has_cat_filter {
			// Indicate any merged cats
			merged_cats := countTmapMergedCatsSpellingVariantsInLinksFromCatFilters(links, cat_filters)
			return model.TmapIndividualSectionWithCatFiltersPage[T]{
				TmapIndividualSectionPage: &model.TmapIndividualSectionPage[T]{
					Links:          links,
					Cats:           cat_counts,
					Pages:          pages,
					NSFWLinksCount: nsfw_links_count,
				},
				MergedCats: merged_cats,
			}, nil
		} else {
			return model.TmapIndividualSectionPage[T]{
				Links:          links,
				Cats:           cat_counts,
				Pages:          pages,
				NSFWLinksCount: nsfw_links_count,
			}, nil
		}
	// All sections
	} else {
		all_tmap_links, err := getAllTmapLinksForOwnerFromOpts[T](tmap_owner, opts)
		if err != nil {
			return nil, err
		}
		submitted := all_tmap_links.Submitted
		starred := all_tmap_links.Starred
		tagged := all_tmap_links.Tagged

		if len(*submitted)+len(*starred)+len(*tagged) == 0 {
			return model.TmapPage[T]{
				TmapSections:   &model.TmapSections[T]{},
				NSFWLinksCount: nsfw_links_count,
			}, nil
		}

		// Get cat counts
		combined_sections := slices.Concat(*submitted, *starred, *tagged)
		cat_counts := getCatCountsFromTmapLinks(
			&combined_sections,
			cat_counts_opts,
		)

		// Limit to top links per section and get SectionsWithMore
		tmap_sections := limitTmapSectionsAndGetLimitedOnes(
			&model.TmapSections[T]{
				Submitted: submitted,
				Starred:   starred,
				Tagged:    tagged,
			},
		)
		tmap_sections.Cats = cat_counts

		if has_cat_filter {
			// Indicate any merged cats
			merged_cats := countTmapMergedCatsSpellingVariantsInLinksFromCatFilters(&combined_sections, cat_filters)
			return model.TmapWithCatFiltersPage[T]{
				TmapPage: &model.TmapPage[T]{
					TmapSections:   tmap_sections,
					NSFWLinksCount: nsfw_links_count,
				},
				MergedCats: merged_cats,
			}, nil
		} else {
			// Profile is only returned on "blank slate" Treasure Map
			// (no cat filters, no particular section)
			var profile *model.Profile
			profile_sql := query.NewTmapProfile(tmap_owner)
			profile, err = scanTmapProfile(profile_sql)
			if err != nil {
				return nil, err
			}

			return model.TmapWithProfilePage[T]{
				Profile: profile,
				TmapPage: &model.TmapPage[T]{
					TmapSections:   tmap_sections,
					NSFWLinksCount: nsfw_links_count,
				},
			}, nil
		}
	}
}

func getTmapLinksQueryBuilderForSectionForOwner(section string, tmap_owner string) (query.TmapLinksQueryBuilder, error) {
	switch section {
	case "submitted":
		return query.NewTmapSubmitted(tmap_owner), nil
	case "starred":
		return query.NewTmapStarred(tmap_owner), nil
	case "tagged":
		return query.NewTmapTagged(tmap_owner), nil
	default:
		return nil, e.ErrInvalidOnlySectionParams
	}
}

func getPageOfIndividualTmapSectionLinks[T model.TmapLink | model.TmapLinkSignedIn](page int, links *[]T) (*[]T, int, error) {
	if page < 0 {
		return nil, 0, e.ErrInvalidPageParams
	} else if page == 0 {
		page = 1
	}

	pages := int(math.Ceil(float64(len(*links)) / float64(query.LINKS_PAGE_LIMIT)))
	if page > pages {
		links = &[]T{}
	} else if page == pages {
		*links = (*links)[query.LINKS_PAGE_LIMIT*(page - 1):]
	} else {
		*links = (*links)[query.LINKS_PAGE_LIMIT*(page - 1) : query.LINKS_PAGE_LIMIT * page]
	}

	return links, pages, nil
}

func getAllTmapLinksForOwnerFromOpts[T model.TmapLink | model.TmapLinkSignedIn](tmap_owner_login_name string, opts *model.TmapOptions) (*struct {
	Submitted *[]T
	Starred   *[]T
	Tagged    *[]T
}, error) {

	submitted, err := buildTmapLinksQueryAndScan[T](
		query.NewTmapSubmitted(tmap_owner_login_name),
		opts,
	)
	if err != nil {
		return nil, err
	}

	starred, err := buildTmapLinksQueryAndScan[T](
		query.NewTmapStarred(tmap_owner_login_name),
		opts,
	)
	if err != nil {
		return nil, err
	}

	tagged, err := buildTmapLinksQueryAndScan[T](
		query.NewTmapTagged(tmap_owner_login_name),
		opts,
	)
	if err != nil {
		return nil, err
	}

	return &struct {
		Submitted *[]T
		Starred   *[]T
		Tagged    *[]T
	}{
		Submitted: submitted,
		Starred:   starred,
		Tagged:    tagged,
	}, nil
}

// sections > LINKS_PAGE_LIMIT links: limit and indicate in response so frontend
// knows to offer pagination
func limitTmapSectionsAndGetLimitedOnes[T model.TmapLink | model.TmapLinkSignedIn](sections *model.TmapSections[T]) *model.TmapSections[T] {
	submitted := sections.Submitted
	starred := sections.Starred
	tagged := sections.Tagged
	var sections_with_more []string

	if len(*submitted) > query.LINKS_PAGE_LIMIT {
		sections_with_more = append(sections_with_more, "submitted")
		*submitted = (*submitted)[0:query.LINKS_PAGE_LIMIT]
	}
	if len(*starred) > query.LINKS_PAGE_LIMIT {
		sections_with_more = append(sections_with_more, "starred")
		*starred = (*starred)[0:query.LINKS_PAGE_LIMIT]
	}
	if len(*tagged) > query.LINKS_PAGE_LIMIT {
		sections_with_more = append(sections_with_more, "tagged")
		*tagged = (*tagged)[0:query.LINKS_PAGE_LIMIT]
	}

	sections.Submitted = submitted
	sections.Starred = starred
	sections.Tagged = tagged
	sections.SectionsWithMore = sections_with_more

	return sections
}

func buildTmapLinksQueryAndScan[T model.TmapLink | model.TmapLinkSignedIn](builder query.TmapLinksQueryBuilder, opts *model.TmapOptions) (*[]T, error) {
	get_links_query := builder.
		FromOptions(opts).
		Build()
	if get_links_query.Error != nil {
		return nil, get_links_query.Error
	}

	links, err := scanTmapLinks[T](get_links_query)
	if err != nil {
		return nil, err
	}

	return links, nil
}

func scanTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](q *query.Query) (*[]T, error) {
	rows, err := q.ValidateAndExecuteRows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links any

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
				&l.TimesStarred,
				&l.AvgStars,
				&l.EarliestStarrers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename)
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
			if err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.CatsFromUser,
				&l.Summary,
				&l.SummaryCount,
				&l.TimesStarred,
				&l.AvgStars,
				&l.EarliestStarrers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,

				// signed-in only
				&l.StarsAssigned,
			); err != nil {
				return nil, err
			}
			signed_in_links = append(signed_in_links, l)
		}

		links = &signed_in_links
	}

	return links.(*[]T), nil
}

// Counting cats and pagination are currently done in Go because merging
// all the links SQL queries together is a headache and doesn't improve
// perf thattt much since tmap contains <= 30 links at a time
// (if LINKS_PAGE_LIMIT is 10, individual sections contain <= 10 links)
func getCatCountsFromTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](links *[]T, opts *model.TmapCatCountsOptions) *[]model.CatCount {
	var omitted_cats []string
	// Use raw_cats_params to determine omitted_cats because CatsFilter
	// (from BuildTmapFromOpts) is modified to escape reserved chars

	if opts != nil && opts.RawCatsParams != "" {
		omitted_cats = strings.Split(strings.ToLower(opts.RawCatsParams), ",")
	}
	has_cat_filter := len(omitted_cats) > 0

	counts := []model.CatCount{}
	all_found_cats := []string{}

	for _, link := range *links {
		var cats_str string
		switch l := any(link).(type) {
		case model.TmapLink:
			cats_str = l.Cats
		case model.TmapLinkSignedIn:
			cats_str = l.Cats
		}

		link_cats := strings.Split(cats_str, ",")
		for i, lc := range link_cats {
			if strings.TrimSpace(lc) == "" {
				continue
			}

			skip := false
			// Skip if resembles a cat filter
			if has_cat_filter {
				for _, oc := range omitted_cats {
					if CatsResembleEachOther(lc, oc) {
						skip = true
					}
				}
			}

			// Skip if resembles another cat from same link.
			// The same link does not need to double-count "book" if
			// it has both "book" and "books."
			if !skip {
				for _, other_lc := range link_cats[i + 1:] {
					if CatsResembleEachOther(lc, other_lc) {
						skip = true
					}
				}
			}

			if !skip {
				// Increment count if existing
				found := false
				for _, found_cat := range all_found_cats {
					if found_cat == lc {
						found = true

						for i, count := range counts {
							if count.Category == lc {
								counts[i].Count++
								break
							}
						}
					}
				}

				// Or create new count
				if !found {
					counts = append(counts, model.CatCount{Category: lc, Count: 1})
					all_found_cats = append(all_found_cats, lc)
				}
			}
		}
	}

	mergeCountsOfCatSpellingVariants(&counts)

	if len(counts) > TMAP_CATS_PAGE_LIMIT {
		counts = (counts)[:TMAP_CATS_PAGE_LIMIT]
	}

	return &counts
}

func mergeCountsOfCatSpellingVariants(counts *[]model.CatCount) {
	// Sort first so most-frequent spelling / casing variants are the ones merged into
	slices.SortFunc(*counts, model.SortCats)

	for i := 0; i < len(*counts); i++ {
		for j := i + 1; j < len(*counts); {
			if CatsResembleEachOther((*counts)[i].Category, (*counts)[j].Category) {
				(*counts)[i].Count += (*counts)[j].Count
				*counts = append((*counts)[:j], (*counts)[j+1:]...)
			} else {
				j++
			}
		}
	}
}

func countTmapMergedCatsSpellingVariantsInLinksFromCatFilters[T model.TmapLink | model.TmapLinkSignedIn](links *[]T, cat_filters []string) []string {
	if links == nil || len(*links) == 0 {
		return nil
	}

	var merged_cats []string
	for _, l := range *links {
		var cats_str string
		switch l := any(l).(type) {
		case model.TmapLink:
			cats_str = l.Cats
		case model.TmapLinkSignedIn:
			cats_str = l.Cats
		}

		link_cats := strings.SplitSeq(cats_str, ",")
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

// USER PROFILE
// (visible on Treasure Map page when no filters applied)
func scanTmapProfile(profile_sql *query.TmapProfile) (*model.Profile, error) {
	var u model.Profile
	if err := db.Client.
		QueryRow(profile_sql.Text, profile_sql.Args...).
		Scan(
			&u.LoginName,
			&u.PFP,
			&u.About,
			&u.Email,
			&u.CreatedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, e.ErrNoUserWithLoginName
		} else {
			return nil, err
		}
	}

	return &u, nil
}

func UserWithIDHasProfilePic(user_id string) bool {
	var p sql.NullString
	if err := db.Client.QueryRow("SELECT pfp FROM Users WHERE id = ?", user_id).Scan(&p); err != nil {
		return false
	}
	return p.Valid
}
