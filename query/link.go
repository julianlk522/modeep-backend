package query

import (
	"fmt"
	"net/url"
	"strings"

	e "github.com/julianlk522/fitm/error"
	mutil "github.com/julianlk522/fitm/model/util"
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
			Args: []any{
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT,
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT,
				LINKS_PAGE_LIMIT,
			},
		},
	})
}

const LINKS_BASE_CTES = `WITH LikeCount AS (
    SELECT link_id, COUNT(*) AS like_count 
    FROM "Link Likes"
    GROUP BY link_id
),
EarliestLikers AS (
    SELECT 
        link_id,
        GROUP_CONCAT(login_name, ', ') AS earliest_likers
    FROM (
        SELECT 
            ll.link_id,
            u.login_name,
            ROW_NUMBER() OVER (PARTITION BY ll.link_id ORDER BY ll.timestamp ASC) as row_num
        FROM "Link Likes" ll
        JOIN Users u ON ll.user_id = u.id
		ORDER BY ll.timestamp ASC, u.login_name ASC
    ) ranked
    WHERE row_num <= ?
    GROUP BY link_id
),
CopyCount AS (
	SELECT link_id, COUNT(*) AS copy_count
	FROM "Link Copies"
	GROUP BY link_id
),
EarliestCopiers AS (
    SELECT 
        link_id,
        GROUP_CONCAT(login_name, ', ') AS earliest_copiers
    FROM (
        SELECT 
            lc.link_id,
            u.login_name,
            ROW_NUMBER() OVER (PARTITION BY lc.link_id ORDER BY lc.timestamp ASC) as row_num
        FROM "Link Copies" lc
        JOIN Users u ON lc.user_id = u.id
		ORDER BY lc.timestamp ASC, u.login_name ASC
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
    COALESCE(lc.like_count, 0) AS like_count,
	COALESCE(el.earliest_likers, '') AS earliest_likers,
	COALESCE(cpc.copy_count, 0) AS copy_count,
	COALESCE(ec.earliest_copiers, '') AS earliest_copiers,
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
LEFT JOIN LikeCount lc ON l.id = lc.link_id
LEFT JOIN EarliestLikers el ON l.id = el.link_id
LEFT JOIN CopyCount cpc ON l.id = cpc.link_id
LEFT JOIN EarliestCopiers ec ON l.id = ec.link_id
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
	// this first because using sort_params value helps with
	// later text replaces since ORDER BY goes at the end
	// (hence the methods below with a "sort_params" arg - it is
	// not the prettiest solution but it works)
	sort_params := params.Get("sort_by")
	if sort_params != "" {
		tl = tl.SortBy(sort_params)
	}

	cats_params := params.Get("cats")
	if cats_params != "" {
		cats := strings.Split(cats_params, ",")
		tl = tl.FromCats(cats)
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
		tl.Error = fmt.Errorf("no cats provided")
		return tl
	}
	// Pop limit arg
	tl.Args = tl.Args[:len(tl.Args)-1]

	// Build and add match arg
	cats = GetCatsOptionalPluralOrSingularForms(cats)
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

// EarliestLikers/Copiers row nums + default NSFW clause makes 3
const NUM_WHERES_IN_BASE_QUERY = 3

func (tl *TopLinks) WithURLContaining(snippet string, sort_by string) *TopLinks {
	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") > NUM_WHERES_IN_BASE_QUERY {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	and_clause := `
	url LIKE ?`

	order_by_clause := LINKS_ORDER_BY
	switch sort_by {
		case "newest":
			order_by_clause = LINKS_ORDER_BY_NEWEST
		case "":
		case "rating":
		default:
			tl.Error = fmt.Errorf("invalid sort_by value")
			return tl
		}

	tl.Text = strings.Replace(
		tl.Text,
		order_by_clause,
		"\n" + clause_keyword + " " + and_clause + order_by_clause,
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
	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") > NUM_WHERES_IN_BASE_QUERY {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	and_clause := `
	url NOT LIKE ?`

	order_by_clause := LINKS_ORDER_BY
	switch sort_by {
		case "newest":
			order_by_clause = LINKS_ORDER_BY_NEWEST
		case "":
		case "rating":
		default:
			tl.Error = fmt.Errorf("invalid sort_by value")
			return tl
		}

	tl.Text = strings.Replace(
		tl.Text,
		order_by_clause,
		"\n" + clause_keyword + " " + and_clause + order_by_clause,
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

	var clause_keyword string
	if strings.Count(tl.Text, "WHERE") > NUM_WHERES_IN_BASE_QUERY {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	order_by_clause := LINKS_ORDER_BY
	if sort_by == "newest" {
		order_by_clause = LINKS_ORDER_BY_NEWEST
	}

	tl.Text = strings.Replace(
		tl.Text,
		order_by_clause,
		"\n" + clause_keyword + " " + period_clause + order_by_clause,
		1,
	)

	return tl
}

func (tl *TopLinks) SortBy(order_by string) *TopLinks {
	if order_by == "newest" {
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_ORDER_BY,
			LINKS_ORDER_BY_NEWEST,
			1,
		)
	} else if order_by != "rating" {
		tl.Error = fmt.Errorf("invalid order_by value")
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

	// insert req_user_id * 2 between 2nd and 3rd args (indexes 1 and 2)
	first_2_args := tl.Args[:2]
	trailing_args := tl.Args[2:]

	new_args := make([]any, 0, len(tl.Args)+2)
	new_args = append(new_args, first_2_args...)
	new_args = append(new_args, req_user_id, req_user_id)
	new_args = append(new_args, trailing_args...)

	tl.Args = new_args

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
	has_subsequent_clause := strings.Contains(
		tl.Text, 
		LINKS_NO_NSFW_CATS_WHERE + "\nAND",
	)
	if has_subsequent_clause {
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_NO_NSFW_CATS_WHERE+"\nAND",
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

func (tl *TopLinks) NSFWLinks(nsfw_params bool) *TopLinks {
	count_select := `
	SELECT count(l.id)`

	// attempt to replace both base and auth-enabled fields
	// it should be one or the other
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_FIELDS + LINKS_AUTH_FIELDS,
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
	if nsfw_params {
		// insert in front of both LINKS_ORDER_BY and LINKS_ORDER_BY_NEWEST
		// (one should be no-op)
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_ORDER_BY,
			NSFW_CLAUSE,
			1,
		)

		tl.Text = strings.Replace(
			tl.Text,
			LINKS_ORDER_BY_NEWEST,
			NSFW_CLAUSE,
			1,
		)
	} else {
		tl.Text = strings.Replace(
			tl.Text,
			LINKS_NO_NSFW_CATS_WHERE,
			`
			WHERE l.id IN (
				SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
			)`,
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

const NSFW_CLAUSE = `WHERE l.id IN (
	SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
)`

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
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT,
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT,
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
LikeCount AS (
    SELECT 
        link_id, 
        COUNT(*) as like_count
    FROM "Link Likes"
    GROUP BY link_id
),
EarliestLikers AS (
    SELECT 
        link_id,
        GROUP_CONCAT(login_name, ', ') AS earliest_likers
    FROM (
        SELECT 
            ll.link_id,
            u.login_name,
            ROW_NUMBER() OVER (PARTITION BY ll.link_id ORDER BY ll.timestamp ASC, u.login_name ASC) as row_num
        FROM "Link Likes" ll
        JOIN Users u ON ll.user_id = u.id
		ORDER BY ll.timestamp ASC, u.login_name ASC
    ) ranked
    WHERE row_num <= ?
    GROUP BY link_id
),
CopyCount AS (
    SELECT 
        link_id, 
        COUNT(*) as copy_count
    FROM "Link Copies"
    GROUP BY link_id
),
EarliestCopiers AS (
    SELECT 
        link_id,
        GROUP_CONCAT(login_name, ', ') AS earliest_copiers
    FROM (
        SELECT 
            lc.link_id,
            u.login_name,
            ROW_NUMBER() OVER (PARTITION BY lc.link_id ORDER BY lc.timestamp ASC, u.login_name ASC) as row_num
        FROM "Link Copies" lc
        JOIN Users u ON lc.user_id = u.id
		ORDER BY lc.timestamp ASC, u.login_name ASC
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
    COALESCE(lc.like_count, 0) as like_count,
    COALESCE(el.earliest_likers, "") as earliest_likers,
    COALESCE(cc.copy_count, 0) as copy_count,
	COALESCE(ec.earliest_copiers, "") as earliest_copiers,
    COALESCE(ckc.click_count, 0) as click_count,
    COALESCE(tc.tag_count, 0) as tag_count,
    b.img_file`

const SINGLE_LINK_FROM = `
FROM Base b`

const SINGLE_LINK_BASE_JOINS = `
LEFT JOIN SummaryCount sc ON sc.link_id = b.link_id
LEFT JOIN LikeCount lc ON lc.link_id = b.link_id
LEFT JOIN EarliestLikers el ON el.link_id = b.link_id
LEFT JOIN CopyCount cc ON cc.link_id = b.link_id
LEFT JOIN EarliestCopiers ec ON ec.link_id = b.link_id
LEFT JOIN ClickCount ckc ON ckc.link_id = b.link_id
LEFT JOIN TagCount tc ON tc.link_id = b.link_id`

func (sl *SingleLink) AsSignedInUser(user_id string) *SingleLink {
	sl.Text = strings.Replace(
		sl.Text,
		SINGLE_LINK_BASE_CTES,
		SINGLE_LINK_BASE_CTES + SINGLE_LINK_AUTH_CTES,
		1,
	)

	sl.Text = strings.Replace(
		sl.Text,
		SINGLE_LINK_BASE_FIELDS,
		SINGLE_LINK_BASE_FIELDS + SINGLE_LINK_AUTH_FIELDS,
		1,
	)

	sl.Text = strings.Replace(
		sl.Text,
		SINGLE_LINK_BASE_JOINS,
		SINGLE_LINK_BASE_JOINS + SINGLE_LINK_AUTH_JOINS,
		1,
	)

	sl.Args = append(sl.Args, user_id, user_id)

	return sl
}

const SINGLE_LINK_AUTH_CTES = `,
IsLiked AS (
    SELECT 
        link_id,
        COUNT(*) as is_liked
    FROM "Link Likes"
    WHERE user_id = ?
    GROUP BY link_id
),
IsCopied AS (
    SELECT 
        link_id,
        COUNT(*) as is_copied
    FROM "Link Copies"
    WHERE user_id = ?
    GROUP BY link_id
)`

const SINGLE_LINK_AUTH_FIELDS = `,
COALESCE(il.is_liked, 0) as is_liked,
COALESCE(ic.is_copied, 0) as is_copied`

const SINGLE_LINK_AUTH_JOINS = `
LEFT JOIN IsLiked il ON il.link_id = b.link_id
LEFT JOIN IsCopied ic ON ic.link_id = b.link_id`