package query

import (
	"fmt"
	"strings"

	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	mutil "github.com/julianlk522/modeep/model/util"
)

type TopLinks struct {
	Query
	selectedSortBy model.SortBy
	// for consistent strings replaces
	hasAndAfterJoins bool
}

func NewTopLinks() *TopLinks {
	return (&TopLinks{
		Query: Query{
			Text: links_base_query,
			Args: []any{
				mutil.EARLIEST_STARRERS_LIMIT,
				LINKS_PAGE_LIMIT,
			},
		},
		// default
		selectedSortBy: model.SortByTimesStarred,
	})
}

func (tl *TopLinks) FromOptions(opts *model.TopLinksOptions) (*TopLinks, error) {
	if opts.SortBy != "" {
		tl = tl.sortBy(opts.SortBy)
	}
	if opts.CatFiltersWithSpellingVariants != nil {
		tl = tl.fromCatFilters(opts.CatFiltersWithSpellingVariants)
	}
	if opts.NeuteredCatFilters != nil {
		tl = tl.fromNeuteredCatFilters(opts.NeuteredCatFilters)
	}
	if opts.GlobalSummaryContains != "" {
		tl = tl.whereGlobalSummaryContains(opts.GlobalSummaryContains)
	}
	if opts.URLContains != "" {
		tl = tl.whereURLContains(opts.URLContains)
	}
	if opts.URLLacks != "" {
		tl = tl.whereURLLacks(opts.URLLacks)
	}
	if opts.Period != "" {
		tl = tl.duringPeriod(opts.Period)
	}
	if opts.AsSignedInUser != "" {
		tl = tl.asSignedInUser(opts.AsSignedInUser)
	}
	if opts.IncludeNSFW {
		tl = tl.includeNSFW()
	}
	if opts.Page != 1 {
		tl = tl.page(opts.Page)
	}
	if tl.Error != nil {
		return nil, tl.Error
	}
	return tl, nil
}

func (tl *TopLinks) sortBy(metric model.SortBy) *TopLinks {
	if metric != "" && metric != model.SortByTimesStarred {
		order_by_clause, ok := links_order_by_clauses[metric]
		if !ok {
			tl.Error = e.ErrInvalidSortByParams
		} else {
			tl.Text = strings.Replace(
				tl.Text,
				LINKS_ORDER_BY_TIMES_STARRED,
				order_by_clause,
				1,
			)
		}
	}

	tl.selectedSortBy = metric
	return tl
}

func (tl *TopLinks) fromCatFilters(cat_filters []string) *TopLinks {
	if len(cat_filters) == 0 || cat_filters[0] == "" {
		tl.Error = e.ErrNoCats
		return tl
	}

	// Add CTE
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_FIELDS,
		",\n"+LINKS_CAT_FILTERS_CTE+LINKS_BASE_FIELDS,
		1,
	)

	// Add JOIN
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_FROM,
		LINKS_FROM+"\n"+LINKS_CAT_FILTERS_JOIN,
		1,
	)

	// Build and add MATCH arg
	// (cats have their singular/plural variant forms added first
	// in FromRequestParams())
	var match_arg = cat_filters[0]
	for i := 1; i < len(cat_filters); i++ {
		match_arg += " AND " + cat_filters[i]
	}

	// Insert before last arg (LIMIT)
	tl.Args = append(tl.Args[:len(tl.Args)-1], match_arg, LINKS_PAGE_LIMIT)

	return tl
}

const LINKS_CAT_FILTERS_CTE = `CatFilters AS (
	SELECT link_id
	FROM global_cats_fts
	WHERE global_cats MATCH ?
)`
const LINKS_CAT_FILTERS_JOIN = `INNER JOIN CatFilters cf ON l.id = cf.link_id`

func (tl *TopLinks) fromNeuteredCatFilters(neutered_cat_filters []string) *TopLinks {
	if len(neutered_cat_filters) == 0 || neutered_cat_filters[0] == "" {
		tl.Error = e.ErrNoCats
		return tl
	}

	// Build IN clause
	in_clause := "WHERE LOWER(global_cat) IN (?"
	for i := 1; i < len(neutered_cat_filters); i++ {
		in_clause += ", ?"
	}
	in_clause += ")"

	// Build CTEs
	neutered_cat_filters_ctes := strings.Replace(
		LINKS_NEUTERED_CAT_FILTERS_CTES,
		"WHERE LOWER(global_cat) IN (?)",
		in_clause,
		1,
	)

	// Add CTEs
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_FIELDS,
		",\n"+neutered_cat_filters_ctes+LINKS_BASE_FIELDS,
		1,
	)

	// Add AND
	selected_order_by_clause := links_order_by_clauses[tl.selectedSortBy]
	tl.Text = strings.Replace(
		tl.Text,
		selected_order_by_clause,
		"\n"+LINKS_NEUTERED_CATS_AND+selected_order_by_clause,
		1,
	)
	// let later methods know
	tl.hasAndAfterJoins = true

	// Add args: {neutered_cat_filters...}
	// Since we use IN, not FTS MATCH, casing matters and spelling variants
	// are not needed.
	neutered_cat_filters_args := make([]any, len(neutered_cat_filters))
	for i, cat := range neutered_cat_filters {
		neutered_cat_filters_args[i] = strings.ToLower(cat)
	}

	// old: [EARLIEST_STARRERS_LIMIT, LINKS_PAGE_LIMIT]
	// new: [EARLIEST_STARRERS_LIMIT, neutered_cat_filters..., LINKS_PAGE_LIMIT]

	// OR if .fromCatFilters() was called first:

	// old: [EARLIEST_STARRERS_LIMIT, cat_filters, LINKS_PAGE_LIMIT]
	// new: [EARLIEST_STARRERS_LIMIT, cat_filters, neutered_cat_filters..., LINKS_PAGE_LIMIT]

	// so can insert 2nd-to-last before LINKS_PAGE_LIMIT
	tl.Args = append(tl.Args[:len(tl.Args)-1], neutered_cat_filters_args...)
	tl.Args = append(tl.Args, LINKS_PAGE_LIMIT)

	return tl
}

const LINKS_NEUTERED_CAT_FILTERS_CTES = `GlobalCatsSplit(link_id, global_cat, str) AS (
    SELECT id, '', global_cats||','
    FROM Links
    UNION ALL SELECT
        link_id,
        substr(str, 0, instr(str, ',')),
        substr(str, instr(str, ',') + 1)
    FROM GlobalCatsSplit
    WHERE str != ''
),
ExcludedLinksDueToNeutering AS (
	SELECT link_id
	FROM GlobalCatsSplit
	WHERE LOWER(global_cat) IN (?)
)`
const LINKS_NEUTERED_CATS_AND = "AND l.id NOT IN ExcludedLinksDueToNeutering"

func (tl *TopLinks) whereGlobalSummaryContains(snippet string) *TopLinks {
	selected_order_by_clause := links_order_by_clauses[tl.selectedSortBy]
	tl.Text = strings.Replace(
		tl.Text,
		selected_order_by_clause,
		// As long as this is called before .includeNSFW() and the
		// LINKS_NO_NSFW_CATS_WHERE clause is still there, this should be an
		// AND.
		"\n"+"AND global_summary LIKE ?"+selected_order_by_clause,
		1,
	)
	tl.hasAndAfterJoins = true

	// insert into args in 2nd-to-last position
	last_arg := tl.Args[len(tl.Args)-1]
	tl.Args = tl.Args[:len(tl.Args)-1]
	tl.Args = append(tl.Args, "%"+snippet+"%")
	tl.Args = append(tl.Args, last_arg)

	return tl
}

func (tl *TopLinks) whereURLContains(snippet string) *TopLinks {
	selected_order_by_clause := links_order_by_clauses[tl.selectedSortBy]
	tl.Text = strings.Replace(
		tl.Text,
		selected_order_by_clause,
		// As long as this is called before .includeNSFW() and the
		// LINKS_NO_NSFW_CATS_WHERE clause is still there, this should be an
		// AND.
		"\n"+"AND url LIKE ?"+selected_order_by_clause,
		1,
	)
	tl.hasAndAfterJoins = true

	// insert into args in 2nd-to-last position
	last_arg := tl.Args[len(tl.Args)-1]
	tl.Args = tl.Args[:len(tl.Args)-1]
	tl.Args = append(tl.Args, "%"+snippet+"%")
	tl.Args = append(tl.Args, last_arg)

	return tl
}

func (tl *TopLinks) whereURLLacks(snippet string) *TopLinks {
	selected_order_by_clause := links_order_by_clauses[tl.selectedSortBy]
	tl.Text = strings.Replace(
		tl.Text,
		selected_order_by_clause,
		// As long as this is called before .includeNSFW() and the
		// LINKS_NO_NSFW_CATS_WHERE clause is still there, this should be an
		// AND.
		"\n"+"AND url NOT LIKE ?"+selected_order_by_clause,
		1,
	)
	tl.hasAndAfterJoins = true

	// insert into args in 2nd-to-last position
	last_arg := tl.Args[len(tl.Args)-1]
	tl.Args = tl.Args[:len(tl.Args)-1]
	tl.Args = append(tl.Args, "%"+snippet+"%")
	tl.Args = append(tl.Args, last_arg)

	return tl
}

func (tl *TopLinks) duringPeriod(period model.Period) *TopLinks {
	if period == "all" {
		return tl
	}
	period_clause, err := getPeriodClause(period)
	if err != nil {
		tl.Error = err
		return tl
	}

	selected_order_by_clause := links_order_by_clauses[tl.selectedSortBy]
	tl.Text = strings.Replace(
		tl.Text,
		selected_order_by_clause,
		// As long as this is called before .includeNSFW() and the
		// LINKS_NO_NSFW_CATS_WHERE clause is still there, this should be an
		// AND.
		"\n"+"AND "+period_clause+selected_order_by_clause,
		1,
	)
	tl.hasAndAfterJoins = true

	return tl
}

func (tl *TopLinks) asSignedInUser(req_user_id string) *TopLinks {
	auth_replacer := strings.NewReplacer(
		LINKS_BASE_CTES, LINKS_BASE_CTES+LINKS_AUTH_CTE,
		LINKS_BASE_FIELDS, LINKS_BASE_FIELDS+LINKS_AUTH_FIELD,
		LINKS_BASE_JOINS, LINKS_BASE_JOINS+LINKS_AUTH_JOIN,
	)
	tl.Text = auth_replacer.Replace(tl.Text)

	first_arg := tl.Args[0]
	trailing_args := tl.Args[1:]

	new_args := make([]any, 0, len(tl.Args)+2)
	new_args = append(new_args, first_arg)
	new_args = append(new_args, req_user_id)
	new_args = append(new_args, trailing_args...)

	tl.Args = new_args

	return tl
}

const LINKS_AUTH_CTE = `,
StarsAssigned AS (
	SELECT link_id, num_stars AS stars_assigned
	FROM Stars
	WHERE user_id = ?
	GROUP BY link_id
)`

const LINKS_AUTH_FIELD = `,
	COALESCE(sa.stars_assigned,0) AS stars_assigned`

const LINKS_AUTH_JOIN = `
	LEFT JOIN StarsAssigned sa ON l.id = sa.link_id`

func (tl *TopLinks) includeNSFW() *TopLinks {
	// e.g.,
	// INNER JOIN ...
	// WHERE l.id NOT IN ( ... )
	// AND ...
	if tl.hasAndAfterJoins {
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_NO_NSFW_CATS_WHERE+"\nAND",
			"\nWHERE",
			1,
		)
		// e.g.,
		// INNER JOIN ...
		// WHERE l.id NOT IN ( ... )
	} else {
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_NO_NSFW_CATS_WHERE,
			"",
			1,
		)
	}

	return tl
}

func (tl *TopLinks) page(page uint) *TopLinks {
	if page < 1 {
		return tl
	}

	// Pop limit arg and replace with limit + 1
	tl.Args = tl.Args[:len(tl.Args)-1]
	tl.Args = append(tl.Args, LINKS_PAGE_LIMIT+1)

	if page > 1 {
		// Add offset
		tl.Text = strings.Replace(
			tl.Text,
			"LIMIT ?",
			"LIMIT ? OFFSET ?",
			1)

		tl.Args = append(tl.Args, (page-1)*LINKS_PAGE_LIMIT)

	}

	return tl
}

func (tl *TopLinks) CountNSFWLinks() *TopLinks {
	count_select := `
	SELECT count(l.id)`

	// Replace either base or base + auth fields
	// (if first works second will be no-op)
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_FIELDS+LINKS_AUTH_FIELD,
		count_select,
		1,
	)
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_FIELDS,
		count_select,
		1,
	)

	// Remove the ORDER BY since this is just a count, after using for
	// string replacement
	selected_order_by_clause := links_order_by_clauses[tl.selectedSortBy]

	// Invert NSFW clause
	var nsfw_clause = `
	WHERE l.id IN (
		SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
	)`

	// If the normal NOT IN nsfw clause is present that can simply be overwritten
	if strings.Contains(
		tl.Text,
		LINKS_NO_NSFW_CATS_WHERE,
	) {
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_NO_NSFW_CATS_WHERE,
			nsfw_clause,
			1,
		)
		// Otherwise confirm whether to say WHERE or AND
	} else {
		// It is possible for there actually not to be an AND after the joins
		// even while this flag is set (if some method sets it and then
		// .includeNSFW() removes the NOT IN nsfw clause and swaps the following
		// AND to WHERE) however in that case we still want to use AND for the
		// nsfw_clause.
		// TODO more intuitive name / approach to this
		if tl.hasAndAfterJoins {
			tl.Text = strings.Replace(
				tl.Text,
				selected_order_by_clause,
				strings.Replace(nsfw_clause, "WHERE", "AND", 1)+selected_order_by_clause,
				1,
			)
		} else {
			tl.Text = strings.Replace(
				tl.Text,
				selected_order_by_clause,
				nsfw_clause+selected_order_by_clause,
				1,
			)
		}
	}

	// Remove ORDER BY
	tl.Text = strings.Replace(
		tl.Text,
		selected_order_by_clause,
		"",
		1,
	)

	// Remove LIMIT and OFFET clause
	// and pop their respective args
	if strings.Contains(
		tl.Text,
		"LIMIT ? OFFSET ?",
	) {
		tl.Text = strings.Replace(
			tl.Text,
			"LIMIT ? OFFSET ?",
			"",
			1,
		)
		tl.Args = tl.Args[:len(tl.Args)-2]
	} else {
		tl.Text = strings.Replace(
			tl.Text,
			"LIMIT ?",
			"",
			1,
		)
		tl.Args = tl.Args[:len(tl.Args)-1]
	}

	return tl
}

const LINKS_BASE_CTES = `WITH TimesStarred AS (
    SELECT link_id, COUNT(*) AS times_starred 
    FROM Stars
    GROUP BY link_id
),
AverageStars AS (
	SELECT link_id, ROUND(AVG(num_stars), 2) AS avg_stars
	FROM Stars
	GROUP BY link_id
),
EarliestStarrers AS (
    SELECT 
        link_id,
        GROUP_CONCAT(login_name, ', ') AS earliest_starrers
    FROM (
        SELECT 
            s.link_id,
            u.login_name,
            ROW_NUMBER() OVER (PARTITION BY s.link_id ORDER BY s.timestamp ASC) as row_num
        FROM Stars s
        JOIN Users u ON s.user_id = u.id
		ORDER BY s.timestamp ASC, u.login_name ASC
    ) ranked
    WHERE row_num <= ?
    GROUP BY link_id
),
ClickCount AS (
	SELECT link_id, count(*) AS click_count
	FROM Clicks
	GROUP BY link_id
),
TagCount AS (
    SELECT link_id, COUNT(*) AS tag_count
    FROM Tags
    GROUP BY link_id
),
SummaryCount AS (
    SELECT link_id, COUNT(*) AS summary_count
    FROM Summaries
    GROUP BY link_id
)`

var LINKS_BASE_FIELDS = fmt.Sprintf(` 
SELECT 
	l.id, 
    l.url, 
    l.submitted_by AS sb, 
    l.submit_date AS sd, 
    COALESCE(l.global_cats, '') AS cats, 
    COALESCE(l.global_summary, '') AS summary, 
    COALESCE(sc.summary_count, 0) AS summary_count,
    COALESCE(ts.times_starred, 0) AS times_starred,
	COALESCE(avs.avg_stars, 0) AS avg_stars,
	COALESCE(es.earliest_starrers, '') AS earliest_starrers,
	COALESCE(clc.click_count, 0) AS click_count, 
    COALESCE(tc.tag_count, 0) AS tag_count,
    COALESCE(l.img_file, '') AS img_file,
	(COUNT(*) OVER() + %d - 1) / %d AS pages`,
	LINKS_PAGE_LIMIT,
	LINKS_PAGE_LIMIT)

const LINKS_FROM = `
FROM
	Links l`

const LINKS_BASE_JOINS = `
LEFT JOIN TimesStarred ts ON l.id = ts.link_id
LEFT JOIN AverageStars avs ON l.id = avs.link_id
LEFT JOIN EarliestStarrers es ON l.id = es.link_id
LEFT JOIN ClickCount clc ON l.id = clc.link_id
LEFT JOIN TagCount tc ON l.id = tc.link_id
LEFT JOIN SummaryCount sc ON l.id = sc.link_id`

const LINKS_NO_NSFW_CATS_WHERE = `
WHERE l.id NOT IN (
	SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
)`

var links_order_by_clauses = map[model.SortBy]string{
	model.SortByTimesStarred: LINKS_ORDER_BY_TIMES_STARRED,
	model.SortByAverageStars: LINKS_ORDER_BY_AVG_STARS,
	model.SortByNewest:       LINKS_ORDER_BY_NEWEST,
	model.SortByOldest:       LINKS_ORDER_BY_OLDEST,
	model.SortByClicks:       LINKS_ORDER_BY_CLICKS,
}

const LINKS_ORDER_BY_TIMES_STARRED = ` 
ORDER BY 
	times_starred DESC, 
	avg_stars DESC,
	click_count DESC,
	tag_count DESC,
	summary_count DESC, 
	submit_date DESC,
	l.id DESC`

const LINKS_ORDER_BY_AVG_STARS = `
ORDER BY 
	avg_stars DESC, 
	times_starred DESC,
	click_count DESC,
	tag_count DESC,
	summary_count DESC, 
	submit_date DESC,
	l.id DESC`

const LINKS_ORDER_BY_NEWEST = `
ORDER BY 
	submit_date DESC, 
	times_starred DESC, 
	avg_stars DESC,
	click_count DESC, 
	tag_count DESC, 
	summary_count DESC, 
	l.id DESC`

const LINKS_ORDER_BY_OLDEST = `
ORDER BY 
	submit_date ASC, 
	times_starred DESC, 
	avg_stars DESC,
	click_count DESC, 
	tag_count DESC, 
	summary_count DESC, 
	l.id DESC`

const LINKS_ORDER_BY_CLICKS = `
ORDER BY 
	click_count DESC, 
	times_starred DESC, 
	avg_stars DESC,
	tag_count DESC, 
	summary_count DESC, 
	l.id DESC`

const LINKS_LIMIT = `
LIMIT ?;`

var links_base_query = LINKS_BASE_CTES +
	LINKS_BASE_FIELDS +
	LINKS_FROM +
	LINKS_BASE_JOINS +
	LINKS_NO_NSFW_CATS_WHERE +
	LINKS_ORDER_BY_TIMES_STARRED +
	LINKS_LIMIT

// SingleLink used on top Tags + Summary pages for a link
type SingleLink struct {
	Query
}

func NewSingleLink(link_id string) *SingleLink {
	return &SingleLink{
		Query{
			Text: SINGLE_LINK_BASE_CTES +
				SINGLE_LINK_BASE_FIELDS +
				SINGLE_LINK_FROM +
				SINGLE_LINK_BASE_JOINS + ";",
			Args: []any{
				link_id,
				mutil.EARLIEST_STARRERS_LIMIT,
			},
		},
	}
}

const SINGLE_LINK_BASE_CTES = `WITH 
Base AS (
    SELECT
        id as link_id,
        url,
        submitted_by as sb,
        submit_date as sd,
        COALESCE(global_cats, "") as cats,
        COALESCE(global_summary, "") as summary,
        COALESCE(img_file, "") as img_file
    FROM Links
    WHERE id = ?
),
SummaryCount AS (
    SELECT 
        link_id, 
        COUNT(*) AS summary_count
    FROM Summaries
    GROUP BY link_id
),
TimesStarred AS (
    SELECT 
        link_id, 
        COUNT(*) as times_starred
    FROM Stars
    GROUP BY link_id
),
AverageStars AS (
	SELECT 
		link_id, 
		ROUND(AVG(num_stars), 2) AS avg_stars
	FROM Stars
	GROUP BY link_id
),
EarliestStarrers AS (
    SELECT 
        link_id,
        GROUP_CONCAT(login_name, ', ') AS earliest_starrers
    FROM (
        SELECT 
            s.link_id,
            u.login_name,
            ROW_NUMBER() OVER (PARTITION BY s.link_id ORDER BY s.timestamp ASC, u.login_name ASC) as row_num
        FROM Stars s
        JOIN Users u ON s.user_id = u.id
		ORDER BY s.timestamp ASC, u.login_name ASC
    ) ranked
    WHERE row_num <= ?
    GROUP BY link_id
),
ClickCount AS (
    SELECT 
        link_id, 
        COUNT(*) AS click_count
    FROM Clicks
    GROUP BY link_id
),
TagCount AS (
    SELECT 
        link_id, 
        COUNT(*) as tag_count
    FROM Tags
    GROUP BY link_id
)`

const SINGLE_LINK_BASE_FIELDS = `
SELECT
    b.link_id,
    b.url,
    b.sb,
    b.sd,
    b.cats,
    b.summary,
    COALESCE(sc.summary_count, 0) as summary_count,
    COALESCE(ts.times_starred, 0) as times_starred,
    COALESCE(avs.avg_stars, 0) as avg_stars,
    COALESCE(es.earliest_starrers, "") as earliest_starrers,
    COALESCE(ckc.click_count, 0) as click_count,
    COALESCE(tc.tag_count, 0) as tag_count,
    b.img_file`

const SINGLE_LINK_FROM = `
FROM Base b`

const SINGLE_LINK_BASE_JOINS = `
LEFT JOIN SummaryCount sc ON sc.link_id = b.link_id
LEFT JOIN TimesStarred ts ON ts.link_id = b.link_id
LEFT JOIN AverageStars avs ON avs.link_id = b.link_id
LEFT JOIN EarliestStarrers es ON es.link_id = b.link_id
LEFT JOIN ClickCount ckc ON ckc.link_id = b.link_id
LEFT JOIN TagCount tc ON tc.link_id = b.link_id`

func (sl *SingleLink) AsSignedInUser(user_id string) *SingleLink {
	sl.Text = strings.Replace(
		sl.Text,
		SINGLE_LINK_BASE_CTES,
		SINGLE_LINK_BASE_CTES+SINGLE_LINK_AUTH_CTE,
		1,
	)

	sl.Text = strings.Replace(
		sl.Text,
		SINGLE_LINK_BASE_FIELDS,
		SINGLE_LINK_BASE_FIELDS+SINGLE_LINK_AUTH_FIELD,
		1,
	)

	sl.Text = strings.Replace(
		sl.Text,
		SINGLE_LINK_BASE_JOINS,
		SINGLE_LINK_BASE_JOINS+SINGLE_LINK_AUTH_JOIN,
		1,
	)

	sl.Args = append(sl.Args, user_id)

	return sl
}

const SINGLE_LINK_AUTH_CTE = `,
StarsAssigned AS (
    SELECT 
        link_id,
        num_stars as stars_assigned
    FROM Stars
    WHERE user_id = ?
    GROUP BY link_id
)`

const SINGLE_LINK_AUTH_FIELD = `,
COALESCE(sa.stars_assigned, 0) as stars_assigned`

const SINGLE_LINK_AUTH_JOIN = `
LEFT JOIN StarsAssigned sa ON sa.stars_assigned = b.link_id`
