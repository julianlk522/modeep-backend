package query

import (
	"fmt"
	"net/url"
	"strings"

	e "github.com/julianlk522/fitm/error"
)

const LINKS_PAGE_LIMIT = 20

type TopLinks struct {
	Query
}

func NewTopLinks() *TopLinks {
	return (&TopLinks{
		Query: Query{
			Text: LINKS_BASE_CTES +
				LINKS_BASE_FIELDS +
				LINKS_FROM +
				LINKS_BASE_JOINS +
				LINKS_NO_NSFW_CATS_WHERE +
				LINKS_ORDER_BY +
				LINKS_LIMIT,
			Args: []any{LINKS_PAGE_LIMIT},
		},
	})
}

const LINKS_BASE_CTES = `WITH LikeCount AS (
    SELECT link_id, COUNT(*) AS like_count 
    FROM "Link Likes"
    GROUP BY link_id
),
CopyCount AS (
	SELECT link_id, COUNT(*) AS copy_count
	FROM "Link Copies"
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
)
`

var LINKS_BASE_FIELDS = fmt.Sprintf(` 
SELECT 
	l.id, 
    l.url, 
    l.submitted_by AS sb, 
    l.submit_date AS sd, 
    COALESCE(l.global_cats, '') AS cats, 
    COALESCE(l.global_summary, '') AS summary, 
    COALESCE(sc.summary_count, 0) AS summary_count,
    COALESCE(lc.like_count, 0) AS like_count,
	COALESCE(cpc.copy_count, 0) AS copy_count,
	COALESCE(clc.click_count, 0) AS click_count, 
    COALESCE(tc.tag_count, 0) AS tag_count,
    COALESCE(l.img_file, '') AS img_file,
	(COUNT(*) OVER() + %d - 1) / %d AS page_count`, 
LINKS_PAGE_LIMIT,
LINKS_PAGE_LIMIT)

const LINKS_FROM = `
FROM
	Links l`

const LINKS_BASE_JOINS = `
LEFT JOIN LikeCount lc ON l.id = lc.link_id
LEFT JOIN CopyCount cpc ON l.id = cpc.link_id
LEFT JOIN ClickCount clc ON l.id = clc.link_id
LEFT JOIN TagCount tc ON l.id = tc.link_id
LEFT JOIN SummaryCount sc ON l.id = sc.link_id
`

const LINKS_NO_NSFW_CATS_WHERE = `
WHERE l.id NOT IN (
	SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
)`

const LINKS_ORDER_BY = ` 
ORDER BY 
    like_count DESC, 
	copy_count DESC,
	click_count DESC,
	tag_count DESC,
    summary_count DESC, 
	submit_date DESC,
    l.id DESC`

const LINKS_ORDER_BY_NEWEST = `
ORDER BY 
	submit_date DESC, 
	like_count DESC, 
	copy_count DESC,
	click_count DESC, 
	tag_count DESC, 
	summary_count DESC, 
	l.id DESC`

const LINKS_LIMIT = `
LIMIT ?;`

func (tl *TopLinks) FromRequestParams(params url.Values) *TopLinks {
	cats_params := params.Get("cats")
	if cats_params != "" {
		cats := strings.Split(cats_params, ",")
		tl = tl.FromCats(cats)
	}

	url_contains_params := params.Get("url_contains")
	if url_contains_params != "" {
		tl = tl.WithURLContaining(url_contains_params)
	}

	period_params := params.Get("period")
	if period_params != "" {
		tl = tl.DuringPeriod(period_params)
	}

	sort_params := params.Get("sort_by")
	if sort_params != "" {
		tl = tl.SortBy(sort_params)
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
		tl.Error = fmt.Errorf("no cats provided")
		return tl
	}
	// Pop limit arg
	tl.Args = tl.Args[:len(tl.Args)-1]

	// Build and add match arg
	cats = GetCatsOptionalPluralOrSingularForms(
		GetCatsWithEscapedReservedChars(cats),
	)

	var match_arg = cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}
	tl.Args = append(tl.Args, match_arg)

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
		LINKS_BASE_CTES+cats_CTE,
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

	// Append limit arg
	tl.Args = append(tl.Args, LINKS_PAGE_LIMIT)

	return tl
}

func (tl *TopLinks) WithURLContaining(snippet string) *TopLinks {
	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") > 1 {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	and_clause := `
	url LIKE ?`

	tl.Text = strings.Replace(
		tl.Text,
		LINKS_ORDER_BY,
		"\n"+clause_keyword+" "+and_clause+LINKS_ORDER_BY,
		1,
	)

	// insert into args in 2nd-to-last position
	last_arg := tl.Args[len(tl.Args)-1]
	tl.Args = tl.Args[:len(tl.Args)-1]
	tl.Args = append(tl.Args, "%"+snippet+"%")
	tl.Args = append(tl.Args, last_arg)

	return tl
}

func (tl *TopLinks) DuringPeriod(period string) *TopLinks {
	period_clause, err := GetPeriodClause(period)
	if err != nil {
		tl.Error = err
		return tl
	}

	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") > 1 {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	tl.Text = strings.Replace(
		tl.Text,
		LINKS_ORDER_BY,
		"\n"+clause_keyword+" "+period_clause+LINKS_ORDER_BY,
		1,
	)

	return tl
}

func (tl *TopLinks) SortBy(order_by string) *TopLinks {
	switch order_by {
	case "rating":
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_ORDER_BY_NEWEST,
			LINKS_ORDER_BY,
			1,
		)
	case "newest":
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_ORDER_BY,
			LINKS_ORDER_BY_NEWEST,
			1,
		)
	default:
		tl.Error = fmt.Errorf("invalid order_by value")
		return tl
	}

	return tl
}

func (tl *TopLinks) AsSignedInUser(req_user_id string) *TopLinks {
	auth_replacer := strings.NewReplacer(
		// Add auth CTEs
		LINKS_BASE_CTES, LINKS_BASE_CTES+LINKS_AUTH_CTES,
		// Add auth fields
		LINKS_BASE_FIELDS, LINKS_BASE_FIELDS+LINKS_AUTH_FIELDS,
		// Add auth joins
		LINKS_BASE_JOINS, LINKS_BASE_JOINS+LINKS_AUTH_JOINS,
	)
	tl.Text = auth_replacer.Replace(tl.Text)

	// Prepend args
	tl.Args = append([]any{req_user_id, req_user_id}, tl.Args...)

	return tl
}

const LINKS_AUTH_CTES = `,
IsLiked AS (
	SELECT link_id, COUNT(*) AS is_liked
	FROM "Link Likes"
	WHERE user_id = ?
	GROUP BY link_id
),
IsCopied AS (
	SELECT link_id, COUNT(*) AS is_copied
	FROM "Link Copies"
	WHERE user_id = ?
	GROUP BY link_id
)`

const LINKS_AUTH_FIELDS = `,
	COALESCE(il.is_liked,0) AS is_liked,
	COALESCE(ic.is_copied,0) AS is_copied`

const LINKS_AUTH_JOINS = `
	LEFT JOIN IsLiked il ON l.id = il.link_id
	LEFT JOIN IsCopied ic ON l.id = ic.link_id`

func (tl *TopLinks) NSFW() *TopLinks {
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_NO_NSFW_CATS_WHERE+"\nAND",
		"\nWHERE",
		1,
	)

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
