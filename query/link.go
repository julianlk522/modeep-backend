package query

import (
	"fmt"
	"net/url"
	"strings"

	e "github.com/julianlk522/modeep/error"
	mutil "github.com/julianlk522/modeep/model/util"
)

type TopLinks struct {
	Query
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
	})
}

func (tl *TopLinks) FromRequestParams(params url.Values) *TopLinks {

	// this first because using sort_params value helps with
	// later text replaces (hence the methods below using a sort_params arg)
	// ORDER BY goes at the end so it's convenient to prepend new clauses before it
	// and the sort_params value indicates exactly which ORDER BY clause to replace
	// not the prettiest solution but it works...
	sort_params := params.Get("sort_by")
	if sort_params != "" {
		tl = tl.SortBy(sort_params)
	}

	cats_params := params.Get("cats")
	if cats_params != "" {
		cats := strings.Split(cats_params, ",")
		tl = tl.FromCats(cats)
	}

	summary_contains_params := params.Get("summary_contains")
	if summary_contains_params != "" {
		tl = tl.WithGlobalSummaryContaining(summary_contains_params, sort_params)
	}

	url_contains_params := params.Get("url_contains")
	if url_contains_params != "" {
		tl = tl.WithURLContaining(url_contains_params, sort_params)
	}

	url_lacks_params := params.Get("url_lacks")
	if url_lacks_params != "" {
		tl = tl.WithURLLacking(url_lacks_params, sort_params)
	}

	period_params := params.Get("period")
	if period_params != "" {
		tl = tl.DuringPeriod(period_params, sort_params)
	}

	var nsfw_params string
	if params.Get("nsfw") != "" {
		nsfw_params = params.Get("nsfw")
	} else if params.Get("NSFW") != "" {
		nsfw_params = params.Get("NSFW")
	}

	if nsfw_params == "true" {
		tl = tl.NSFW()
	} else if nsfw_params != "false" && nsfw_params != "" {
		tl.Error = e.ErrInvalidNSFWParams
	}

	return tl
}

func (tl *TopLinks) FromCats(cats []string) *TopLinks {
	if len(cats) == 0 || cats[0] == "" {
		tl.Error = e.ErrNoCats
		return tl
	}

	// Build CTE from match_clause
	match_clause := `
	WHERE global_cats MATCH ?`
	cats_CTE := `,
		CatsFilter AS (
			SELECT link_id
			FROM global_cats_fts` + match_clause + `
		)`

	// Prepend CTE
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_CTES,
		LINKS_BASE_CTES + cats_CTE,
		1)

	// Append join
	const LINKS_CATS_JOIN = `
	INNER JOIN CatsFilter f ON l.id = f.link_id`
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_FROM,
		LINKS_FROM+LINKS_CATS_JOIN,
		1,
	)

	// Pop limit arg
	tl.Args = tl.Args[:len(tl.Args)-1]

	// Build and add match arg
	cats = GetCatsOptionalPluralOrSingularForms(cats)
	var match_arg = cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}
	tl.Args = append(tl.Args, match_arg)

	// Re-add limit
	tl.Args = append(tl.Args, LINKS_PAGE_LIMIT)

	return tl
}

func (tl *TopLinks) WithGlobalSummaryContaining(snippet string, sort_by string) *TopLinks {
	order_by_clause := LINKS_ORDER_BY_TIMES_STARRED
	if sort_by != "" {
		clause, ok := links_order_by_clauses[sort_by]
		if ok {
			order_by_clause = clause
		} else {
			tl.Error = e.ErrInvalidSortByParams
			return tl
		}
	}

	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") >= num_wheres_in_links_base_query {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	tl.Text = strings.Replace(
		tl.Text,
		order_by_clause,
		"\n" + clause_keyword + " global_summary LIKE ?" + order_by_clause,
		1,
	)

	// insert into args in 2nd-to-last position
	last_arg := tl.Args[len(tl.Args) - 1]
	tl.Args = tl.Args[:len(tl.Args) - 1]
	tl.Args = append(tl.Args, "%" + snippet + "%")
	tl.Args = append(tl.Args, last_arg)

	return tl
}

func (tl *TopLinks) WithURLContaining(snippet string, sort_by string) *TopLinks {
	order_by_clause := LINKS_ORDER_BY_TIMES_STARRED
	if sort_by != "" {
		clause, ok := links_order_by_clauses[sort_by]
		if ok {
			order_by_clause = clause
		} else {
			tl.Error = e.ErrInvalidSortByParams
			return tl
		}
	}

	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") >= num_wheres_in_links_base_query {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	tl.Text = strings.Replace(
		tl.Text,
		order_by_clause,
		"\n" + clause_keyword + " url LIKE ?" + order_by_clause,
		1,
	)

	// insert into args in 2nd-to-last position
	last_arg := tl.Args[len(tl.Args) - 1]
	tl.Args = tl.Args[:len(tl.Args) - 1]
	tl.Args = append(tl.Args, "%" + snippet + "%")
	tl.Args = append(tl.Args, last_arg)

	return tl
}

func (tl *TopLinks) WithURLLacking(snippet string, sort_by string) *TopLinks {
	order_by_clause := LINKS_ORDER_BY_TIMES_STARRED
	if sort_by != "" {
		clause, ok := links_order_by_clauses[sort_by]
		if ok {
			order_by_clause = clause
		} else {
			tl.Error = e.ErrInvalidSortByParams
			return tl
		}
	}

	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") >= num_wheres_in_links_base_query {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	tl.Text = strings.Replace(
		tl.Text,
		order_by_clause,
		"\n" + clause_keyword + " url NOT LIKE ?" + order_by_clause,
		1,
	)

	// insert into args in 2nd-to-last position
	last_arg := tl.Args[len(tl.Args) - 1]
	tl.Args = tl.Args[:len(tl.Args) - 1]
	tl.Args = append(tl.Args, "%" + snippet + "%")
	tl.Args = append(tl.Args, last_arg)

	return tl
}

func (tl *TopLinks) DuringPeriod(period string, sort_by string) *TopLinks {
	if (period == "all") {
		return tl
	}
	
	period_clause, err := GetPeriodClause(period)
	if err != nil {
		tl.Error = err
		return tl
	}

	order_by_clause := LINKS_ORDER_BY_TIMES_STARRED
	if sort_by != "" {
		clause, ok := links_order_by_clauses[sort_by]
		if ok {
			order_by_clause = clause
		} else {
			tl.Error = e.ErrInvalidSortByParams
			return tl
		}
	}
	
	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") >= num_wheres_in_links_base_query {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	tl.Text = strings.Replace(
		tl.Text,
		order_by_clause,
		"\n" + clause_keyword + " " + period_clause + order_by_clause,
		1,
	)

	return tl
}

var num_wheres_in_links_base_query = strings.Count(links_base_query, "WHERE")

func (tl *TopLinks) SortBy(metric string) *TopLinks {
	if metric != "" && metric != "times_starred" {
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

	return tl
}

func (tl *TopLinks) AsSignedInUser(req_user_id string) *TopLinks {
	auth_replacer := strings.NewReplacer(
		LINKS_BASE_CTES, LINKS_BASE_CTES + LINKS_AUTH_CTE,
		LINKS_BASE_FIELDS, LINKS_BASE_FIELDS + LINKS_AUTH_FIELD,
		LINKS_BASE_JOINS, LINKS_BASE_JOINS + LINKS_AUTH_JOIN,
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

func (tl *TopLinks) NSFW() *TopLinks {
	has_subsequent_clause := strings.Contains(
		tl.Text, 
		LINKS_NO_NSFW_CATS_WHERE + "\nAND",
	)
	if has_subsequent_clause {
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_NO_NSFW_CATS_WHERE + "\nAND",
			"\nWHERE",
			1,
		)
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

func (tl *TopLinks) Page(page int) *TopLinks {
	if page == 0 {
		return tl
	}

	if page >= 1 {
		// Pop limit arg and replace with limit + 1
		tl.Args = append(tl.Args[:len(tl.Args)-1], LINKS_PAGE_LIMIT+1)
	}

	if page == 1 {
		return tl
	}

	tl.Text = strings.Replace(
		tl.Text,
		"LIMIT ?",
		"LIMIT ? OFFSET ?",
		1)

	// Append offset arg
	tl.Args = append(tl.Args, (page-1)*LINKS_PAGE_LIMIT)

	return tl
}

func (tl *TopLinks) CountNSFWLinks(nsfw_params bool) *TopLinks {
	count_select := `
	SELECT count(l.id)`

	// replace either base or auth-enabled fields
	// one will work, other will no-op
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_FIELDS + LINKS_AUTH_FIELD,
		count_select,
		1,
	)

	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_FIELDS,
		count_select,
		1,
	)

	// invert NSFW clause
	var nsfw_clause = `
	WHERE l.id IN (
		SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
	)`

	if strings.Contains(
		tl.Text, 
		"WHERE url",
	) || strings.Contains(
		tl.Text, 
		"WHERE global_summmary",
	) {
		nsfw_clause = strings.Replace(
			nsfw_clause,
			"WHERE",
			"AND",
			1,
		)
	}

	if nsfw_params {
		// insert in front of all LINKS_ORDER_BY variants
		// (all except one should be no-op)
		for _, order_by_clause := range links_order_by_clauses {
			tl.Text = strings.Replace(
				tl.Text,
				order_by_clause,
				nsfw_clause,
				1,
			)
		}
	} else {
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_NO_NSFW_CATS_WHERE,
			nsfw_clause,
			1,
		)
	}

	// remove LIMIT and OFFET clause
	tl.Text = strings.Replace(
		tl.Text,
		"\nLIMIT ? OFFSET ?",
		"",
		1,
	)

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

var links_order_by_clauses = map[string]string{
	"times_starred": LINKS_ORDER_BY_TIMES_STARRED,
	"avg_stars": LINKS_ORDER_BY_AVG_STARS,
	"newest": LINKS_ORDER_BY_NEWEST,
	"oldest": LINKS_ORDER_BY_OLDEST,
	"clicks": LINKS_ORDER_BY_CLICKS,
}

const LINKS_ORDER_BY_TIMES_STARRED = ` 
ORDER BY 
    times_starred DESC, 
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
	click_count DESC, 
	tag_count DESC, 
	summary_count DESC, 
	l.id DESC`

const LINKS_ORDER_BY_OLDEST = `
ORDER BY 
	submit_date ASC, 
	times_starred DESC, 
	click_count DESC, 
	tag_count DESC, 
	summary_count DESC, 
	l.id DESC`

const LINKS_ORDER_BY_CLICKS = `
ORDER BY 
	click_count DESC, 
	times_starred DESC, 
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
		SINGLE_LINK_BASE_CTES + SINGLE_LINK_AUTH_CTE,
		1,
	)

	sl.Text = strings.Replace(
		sl.Text,
		SINGLE_LINK_BASE_FIELDS,
		SINGLE_LINK_BASE_FIELDS + SINGLE_LINK_AUTH_FIELD,
		1,
	)

	sl.Text = strings.Replace(
		sl.Text,
		SINGLE_LINK_BASE_JOINS,
		SINGLE_LINK_BASE_JOINS + SINGLE_LINK_AUTH_JOIN,
		1,
	)

	sl.Args = append(sl.Args, user_id, user_id)

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