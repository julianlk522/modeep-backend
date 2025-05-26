package handler

import (
	"database/sql"
	"math"
	"net/url"
	"strconv"

	"github.com/julianlk522/fitm/db"

	"slices"
	"strings"

	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

const TMAP_CATS_PAGE_LIMIT int = 20

func GetTmapOptsFromRequestParams(params url.Values) (*model.TmapOptions, error) {
	var opts = &model.TmapOptions{}

	cats_params := params.Get("cats")
	if cats_params != "" {

		// For GetCatCountsFromTmapLinks()
		opts.RawCatsParams = cats_params

		cats := query.GetCatsOptionalPluralOrSingularForms(
			query.GetCatsWithEscapedReservedChars(
				strings.Split(cats_params, ","),
			),
		)
		opts.CatsFilter = cats
	}

	var nsfw_params string
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

	sort_params := params.Get("sort_by")
	if sort_params == "newest" {
		opts.SortByNewest = true
	} else if sort_params != "rating" && sort_params != "" {
		return nil, e.ErrInvalidSortByParams
	}

	section_params := strings.ToLower(params.Get("section"))
	if section_params != "" {
		switch section_params {
		case "submitted", "copied", "tagged":
			opts.Section = section_params
		default:
			return nil, e.ErrInvalidSectionParams
		}
	}

	page_params := params.Get("page")
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

	nsfw_links_count_sql := query.NewTmapNSFWLinksCount(tmap_owner)

	has_cat_filter := len(opts.CatsFilter) > 0
	var profile *model.Profile
	if has_cat_filter {
		nsfw_links_count_sql = nsfw_links_count_sql.FromCats(opts.CatsFilter)
		// add profile only if unfiltered
	} else {
		var err error
		profile_sql := query.NewTmapProfile(tmap_owner)
		profile, err = ScanTmapProfile(profile_sql)
		if err != nil {
			return nil, err
		}
	}

	var nsfw_links_count int

	var cat_counts *[]model.CatCount
	var cat_counts_opts *model.TmapCatCountsOptions
	if has_cat_filter {
		cat_counts_opts = &model.TmapCatCountsOptions{
			RawCatsParams: opts.RawCatsParams,
		}
	}

	// Single section
	if opts.Section != "" {
		var links *[]T
		var err error

		switch opts.Section {
		case "submitted":
			links, err = ScanTmapLinks[T](query.
				NewTmapSubmitted(tmap_owner).
				FromOptions(opts).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.SubmittedOnly()
		case "copied":
			links, err = ScanTmapLinks[T](query.
					NewTmapCopied(tmap_owner).
					FromOptions(opts).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.CopiedOnly()
		case "tagged":
			links, err = ScanTmapLinks[T](query.
					NewTmapTagged(tmap_owner).
					FromOptions(opts).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.TaggedOnly()
		default:
			return nil, e.ErrInvalidSectionParams
		}

		if err != nil {
			return nil, err
		}

		if links == nil || len(*links) == 0 {
			return model.TmapSectionPage[T]{
				Links:          &[]T{},
				Cats:           &[]model.CatCount{},
				NSFWLinksCount: 0,
				PageCount:       -1,
			}, nil
		}

		// counting cats and pagination are both done in Go
		// because merging all the links SQL queries together is a headache
		// and doesn't make perf thattt much better since tmap contains <= 60
		// links at a time (single section contains <= 20 links)
		// and looping over <= 60 links is trivial
		cat_counts = GetCatCountsFromTmapLinks(links, cat_counts_opts)

		// Pagination
		// TODO: move to separate util function
		page := 1
		if opts.Page < 0 {
			return nil, e.ErrInvalidPageParams
		} else if opts.Page > 0 {
			page = opts.Page
		}

		page_count := int(math.Ceil(float64(len(*links)) / float64(query.LINKS_PAGE_LIMIT)))
		if page > page_count {
			links = &[]T{}
		} else if page == page_count {
			*links = (*links)[query.LINKS_PAGE_LIMIT*(page-1) : ]
		} else {
			*links = (*links)[query.LINKS_PAGE_LIMIT*(page-1) : query.LINKS_PAGE_LIMIT*page]
		}

		if err := db.Client.QueryRow(
			nsfw_links_count_sql.Text, 
			nsfw_links_count_sql.Args...,
		).Scan(&nsfw_links_count); err != nil {
			return nil, err
		}

		return model.TmapSectionPage[T]{
			Links:          links,
			Cats:           cat_counts,
			PageCount:      page_count,
			NSFWLinksCount: nsfw_links_count,
		}, nil

		// All sections
	} else {
		submitted, err := ScanTmapLinks[T](query.
			NewTmapSubmitted(tmap_owner).
			FromOptions(opts).Query)
		if err != nil {
			return nil, err
		}
		copied, err := ScanTmapLinks[T](query.
			NewTmapCopied(tmap_owner).
			FromOptions(opts).Query)
		if err != nil {
			return nil, err
		}
		tagged, err := ScanTmapLinks[T](query.
			NewTmapTagged(tmap_owner).
			FromOptions(opts).Query)
		if err != nil {
			return nil, err
		}

		links_from_all_sections := slices.Concat(*submitted, *copied, *tagged)
		if len(links_from_all_sections) == 0 {
			return model.FilteredTmap[T]{
				TmapSections:   &model.TmapSections[T]{},
				NSFWLinksCount: 0,
			}, nil
		}

		cat_counts = GetCatCountsFromTmapLinks(&links_from_all_sections, cat_counts_opts)

		// limit sections to top 20 links
		// 20+ links: indicate in response so can be paginated
		var sections_with_more []string
		if len(*submitted) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "submitted")
			*submitted = (*submitted)[0:query.LINKS_PAGE_LIMIT]
		}
		if len(*copied) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "copied")
			*copied = (*copied)[0:query.LINKS_PAGE_LIMIT]
		}
		if len(*tagged) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "tagged")
			*tagged = (*tagged)[0:query.LINKS_PAGE_LIMIT]
		}

		sections := &model.TmapSections[T]{
			Submitted:        submitted,
			Copied:           copied,
			Tagged:           tagged,
			SectionsWithMore: sections_with_more,
			Cats:             cat_counts,
		}

		if err := db.Client.QueryRow(
			nsfw_links_count_sql.Text, 
			nsfw_links_count_sql.Args...,
		).Scan(&nsfw_links_count); err != nil {
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

func ScanTmapProfile(profile_sql *query.TmapProfile) (*model.Profile, error) {
	var u model.Profile
	if err := db.Client.
		QueryRow(profile_sql.Text, profile_sql.Args...).
		Scan(
			&u.LoginName,
			&u.PFP,
			&u.About,
			&u.Email,
			&u.Created,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, e.ErrNoUserWithLoginName
		} else {
			return nil, err
		}
	}

	return &u, nil
}

func ScanTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](sql *query.Query) (*[]T, error) {
	rows, err := db.Client.Query(sql.Text, sql.Args...)
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
				&l.LikeCount,
				&l.EarliestLikers,
				&l.CopyCount,
				&l.EarliestCopiers,
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
				&l.EarliestLikers,
				&l.CopyCount,
				&l.EarliestCopiers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,

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

func GetCatCountsFromTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](links *[]T, opts *model.TmapCatCountsOptions) *[]model.CatCount {
	var omitted_cats []string
	// Use raw_cats_params here to determine omitted_cats because CatsFilter
	// (from BuildTmapFromOpts) is modified to escape reserved chars and
	// include singular/plural spelling variations. To correctly count cats
	// (omitting ones passed in the request), omitted_cats must _not_ have
	// these modifications applied.

	// Use lowercase so that capitalization variants of cat filters
	// are still not counted
	if opts != nil && opts.RawCatsParams != "" {
		omitted_cats = strings.Split(strings.ToLower(opts.RawCatsParams), ",")
	}
	has_cat_filter := len(omitted_cats) > 0

	counts := []model.CatCount{}
	all_found_cats := []string{}
	var found bool

	for _, link := range *links {
		var cats string
		switch l := any(link).(type) {
		case model.TmapLink:
			cats = l.Cats
		case model.TmapLinkSignedIn:
			cats = l.Cats
		}

		link_found_cats := []string{}

		for _, cat := range strings.Split(cats, ",") {
			lc_cat := strings.ToLower(cat)

			if strings.TrimSpace(cat) == "" || slices.ContainsFunc(link_found_cats, func(c string) bool { return strings.ToLower(c) == lc_cat }) || (has_cat_filter &&
				slices.Contains(omitted_cats, lc_cat)) {
				continue
			}

			link_found_cats = append(link_found_cats, cat)

			found = false
			for _, found_cat := range all_found_cats {
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
				all_found_cats = append(all_found_cats, cat)
			}
		}
	}

	slices.SortFunc(counts, model.SortCats)

	if has_cat_filter {
		MergeCatCountsCapitalizationVariants(&counts, omitted_cats)
	}

	if len(counts) > TMAP_CATS_PAGE_LIMIT {
		counts = (counts)[:TMAP_CATS_PAGE_LIMIT]
	}

	return &counts
}

func MergeCatCountsCapitalizationVariants(counts *[]model.CatCount, omitted_cats []string) {
	// e.g. "Music" and "music"
	
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
