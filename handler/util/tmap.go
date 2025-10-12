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

// DeleteProfilePic
func UserWithIDHasProfilePic(user_id string) bool {
	var p sql.NullString
	if err := db.Client.QueryRow("SELECT pfp FROM Users WHERE id = ?", user_id).Scan(&p); err != nil {
		return false
	}
	return p.Valid
}

func GetTmapOptsFromRequestParams(params url.Values) (*model.TmapOptions, error) {
	var opts = &model.TmapOptions{}
	var cats_params, 
		period_params, 
		summary_contains_params,
		url_contains_params, 
		url_lacks_params, 
		nsfw_params, 
		sort_params, 
		section_params, 
		page_params string

	cats_params = params.Get("cats")
	if cats_params != "" {
		// For GetCatCountsFromTmapLinks()
		opts.RawCatsParams = cats_params

		cats := query.GetCatsOptionalPluralOrSingularForms(
			strings.Split(cats_params, ","),
		)
		opts.Cats = cats
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

	// Regardless of individual or multiple sections, cat filters or not, we
	// always return counts for NSFW links and top cats/subcats
	var nsfw_links_count int
	nsfw_links_count_opts := &model.TmapNSFWLinksCountOptions{
		OnlySection:     opts.Section,
		CatsFilter:      opts.Cats,
		Period:          opts.Period,
		SummaryContains: opts.SummaryContains,
		URLContains:     opts.URLContains,
		URLLacks:        opts.URLLacks,
	}
	nsfw_links_count_sql := query.
		NewTmapNSFWLinksCount(tmap_owner).
		FromOptions(nsfw_links_count_opts)
	if err := db.Client.QueryRow(
		nsfw_links_count_sql.Text,
		nsfw_links_count_sql.Args...,
	).Scan(&nsfw_links_count); err != nil {
		return nil, err
	}

	var cat_counts *[]model.CatCount
	var cat_counts_opts *model.TmapCatCountsOptions

	var cat_filters []string
	has_cat_filter := len(opts.Cats) > 0
	if has_cat_filter {
		cat_counts_opts = &model.TmapCatCountsOptions{
			RawCatsParams: opts.RawCatsParams,
		}

		cat_filters = strings.Split(opts.RawCatsParams, ",")
	}

	// Individual section
	if opts.Section != "" {
		var links *[]T
		var err error

		switch opts.Section {
		case "submitted":
			submitted_sql := query.
				NewTmapSubmitted(tmap_owner).
				FromOptions(opts)
			if submitted_sql.Error != nil {
				return nil, submitted_sql.Error
			}

			links, err = ScanTmapLinks[T](submitted_sql.Query)
		case "starred":
			starred_sql := query.
				NewTmapStarred(tmap_owner).
				FromOptions(opts)
			if starred_sql.Error != nil {
				return nil, starred_sql.Error
			}

			links, err = ScanTmapLinks[T](starred_sql.Query)
		case "tagged":
			tagged_sql := query.
				NewTmapTagged(tmap_owner).
				FromOptions(opts)
			if tagged_sql.Error != nil {
				return nil, tagged_sql.Error
			}

			links, err = ScanTmapLinks[T](tagged_sql.Query)
		default:
			return nil, e.ErrInvalidSectionParams
		}

		if err != nil {
			return nil, err
		}

		if links == nil || len(*links) == 0 {
			return model.TmapIndividualSectionPage[T]{
				Links:          &[]T{},
				Cats:           &[]model.CatCount{},
				NSFWLinksCount: nsfw_links_count,
				Pages:          -1,
			}, nil
		}

		// counting cats and pagination are done in Go because merging 
		// all the links SQL queries together is a headache and doesn't make 
		// perf thattt much better since tmap contains <= 60 links at a time 
		// (single section contains <= 20 links)
		cat_counts = GetCatCountsFromTmapLinks(links, cat_counts_opts)

		// Pagination
		// TODO: move to separate util function
		page := 1
		if opts.Page < 0 {
			return nil, e.ErrInvalidPageParams
		} else if opts.Page > 0 {
			page = opts.Page
		}

		pages := int(math.Ceil(float64(len(*links)) / float64(query.LINKS_PAGE_LIMIT)))
		if page > pages {
			links = &[]T{}
		} else if page == pages {
			*links = (*links)[query.LINKS_PAGE_LIMIT*(page - 1):]
		} else {
			*links = (*links)[query.LINKS_PAGE_LIMIT*(page - 1) : query.LINKS_PAGE_LIMIT * page]
		}

		// Indicate any merged cats
		if has_cat_filter {
			merged_cats := GetMergedCatsSpellingVariantsFromTmapLinksWithCatFilters(links, cat_filters)
			return model.TmapIndividualSectionPageWithCatFilters[T]{
				TmapIndividualSectionPage: &model.TmapIndividualSectionPage[T]{
					Links:          links,
					Cats:           cat_counts,
					Pages:          pages,
					NSFWLinksCount: nsfw_links_count,
				},
				MergedCats:     merged_cats,
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
		submitted_sql := query.
			NewTmapSubmitted(tmap_owner).
			FromOptions(opts)
		if submitted_sql.Error != nil {
			return nil, submitted_sql.Error
		}

		submitted, err := ScanTmapLinks[T](submitted_sql.Query)
		if err != nil {
			return nil, err
		}

		starred_sql := query.
			NewTmapStarred(tmap_owner).
			FromOptions(opts)
		if starred_sql.Error != nil {
			return nil, starred_sql.Error
		}

		starred, err := ScanTmapLinks[T](starred_sql.Query)
		if err != nil {
			return nil, err
		}
		
		tagged_sql := query.
			NewTmapTagged(tmap_owner).
			FromOptions(opts)
		if tagged_sql.Error != nil {
			return nil, tagged_sql.Error
		}

		tagged, err := ScanTmapLinks[T](tagged_sql.Query)
		if err != nil {
			return nil, err
		}

		if len(*submitted)+len(*starred)+len(*tagged) == 0 {
			return model.Tmap[T]{
				TmapSections:   &model.TmapSections[T]{},
				NSFWLinksCount: nsfw_links_count,
			}, nil
		}

		// Get cat counts BEFORE pagination so totals still include
		// cats from any links that were excluded
		combined_sections := slices.Concat(*submitted, *starred, *tagged)
		cat_counts = GetCatCountsFromTmapLinks(
			&combined_sections,
			cat_counts_opts,
		)

		// limit to top 20 links / section
		// sections > 20 links: indicate in response so can be paginated
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

		tmap_sections := &model.TmapSections[T]{
			Submitted:        submitted,
			Starred:          starred,
			Tagged:           tagged,
			SectionsWithMore: sections_with_more,
			Cats:             cat_counts,
		}

		// Indicate any merged cats
		if has_cat_filter {
			merged_cats := GetMergedCatsSpellingVariantsFromTmapLinksWithCatFilters(&combined_sections, cat_filters)
			return model.TmapWithCatFilters[T]{
				Tmap: &model.Tmap[T]{
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
			profile, err = ScanTmapProfile(profile_sql)
			if err != nil {
				return nil, err
			}

			return model.TmapWithProfile[T]{
				Profile: profile,
				Tmap: &model.Tmap[T]{
					TmapSections:   tmap_sections,
					NSFWLinksCount: nsfw_links_count,
				},
			}, nil
		}
	}
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

func GetCatCountsFromTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](links *[]T, opts *model.TmapCatCountsOptions) *[]model.CatCount {
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
			
			// skip if resembles a cat filter
			skip := false
			if has_cat_filter {
				for _, oc := range omitted_cats {
					if strings.EqualFold(lc, oc) ||
					CatsAreSingularOrPluralVariationsOfEachOther(lc, oc) {
						skip = true
					}
				}
			}

			// skip if resembles another cat from same link
			if !skip {
				for _, other_lc := range link_cats[i + 1:] {
					if strings.EqualFold(lc, other_lc) ||
					CatsAreSingularOrPluralVariationsOfEachOther(lc, other_lc) {
						skip = true
					}
				}
			}

			if !skip {
				// increment count if existing
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
	
				// or create new count
				if !found {
					counts = append(counts, model.CatCount{Category: lc, Count: 1})
					all_found_cats = append(all_found_cats, lc)
				}
			}
		}
	}

	MergeCatCountsSpellingVariants(&counts)

	if len(counts) > TMAP_CATS_PAGE_LIMIT {
		counts = (counts)[:TMAP_CATS_PAGE_LIMIT]
	}

	return &counts
}

func MergeCatCountsSpellingVariants(counts *[]model.CatCount) {
	// sort to ensure most-frequent spelling / casing variants are used
	slices.SortFunc(*counts, model.SortCats)

	for i := 0; i < len(*counts); i++ {
		for j := i + 1; j < len(*counts); {
			if strings.EqualFold((*counts)[i].Category, (*counts)[j].Category) ||
				CatsAreSingularOrPluralVariationsOfEachOther((*counts)[i].Category, (*counts)[j].Category) {
				(*counts)[i].Count += (*counts)[j].Count
				*counts = append((*counts)[:j], (*counts)[j+1:]...)
			} else {
				j++
			}
		}
	}
}

func GetMergedCatsSpellingVariantsFromTmapLinksWithCatFilters[T model.TmapLink | model.TmapLinkSignedIn](links *[]T, cat_filters []string) []string {
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

					if (CatsAreSingularOrPluralVariationsOfEachOther(cat_lc, cf_lc) || 
					// capitalization variants
					cat_lc == cf_lc && cat != cf) && !slices.Contains(merged_cats, cat) {
						merged_cats = append(merged_cats, cat)
					}
				}
			}

		}

	}
	return merged_cats
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

