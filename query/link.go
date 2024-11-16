package query

import (
	"fmt"
	"strings"
)

const LINKS_PAGE_LIMIT = 20

type TopLinks struct {
	Query
}

func NewTopLinks() *TopLinks {
	return (&TopLinks{
		Query: Query{
			Text: 
				LINKS_BASE_CTES +
				LINKS_BASE_FIELDS +
				LINKS_FROM +
				LINKS_BASE_JOINS +
				LINKS_NO_NSFW_CATS_WHERE +
				LINKS_ORDER_BY +
				LINKS_LIMIT,
			Args: []interface{}{LINKS_PAGE_LIMIT},
		},
	})
}

const LINKS_BASE_CTES = `WITH LikeCount AS (
    SELECT link_id, COUNT(*) AS like_count 
    FROM "Link Likes"
    GROUP BY link_id
),
SummaryCount AS (
    SELECT link_id, COUNT(*) AS summary_count
    FROM Summaries
    GROUP BY link_id
),
TagCount AS (
    SELECT link_id, COUNT(*) AS tag_count
    FROM Tags
    GROUP BY link_id
)`

const LINKS_BASE_FIELDS = ` 
SELECT 
	l.id, 
    l.url, 
    l.submitted_by AS sb, 
    l.submit_date AS sd, 
    COALESCE(l.global_cats, '') AS cats, 
    COALESCE(l.global_summary, '') AS summary, 
    COALESCE(s.summary_count, 0) AS summary_count,
    COALESCE(t.tag_count, 0) AS tag_count,
    COALESCE(ll.like_count, 0) AS like_count, 
    COALESCE(l.img_url, '') AS img_url`

const LINKS_FROM = `
FROM
	Links l`

const LINKS_BASE_JOINS = `
LEFT JOIN LikeCount ll ON l.id = ll.link_id
LEFT JOIN SummaryCount s ON l.id = s.link_id
LEFT JOIN TagCount t ON l.id = t.link_id`

const LINKS_NO_NSFW_CATS_WHERE = `
WHERE l.id NOT IN (
	SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
)`

const LINKS_ORDER_BY = ` 
ORDER BY 
    like_count DESC, 
    summary_count DESC, 
    l.id DESC`

const LINKS_LIMIT = `
LIMIT ?;`

func (tl *TopLinks) FromCats(cats []string) *TopLinks {
	if len(cats) == 0 || cats[0] == "" {
		tl.Error = fmt.Errorf("no cats provided")
		return tl
	}
	// pop limit arg
	tl.Args = tl.Args[:len(tl.Args)-1]

	
	// build and add match arg
	EscapeCatsReservedChars(cats)
	cats = GetCatsOptionalPluralOrSingularForms(cats)

	var match_arg = cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}
	tl.Args = append(tl.Args, match_arg)

	// build CTE from match_clause
	match_clause := `
	WHERE global_cats MATCH ?`
	cats_CTE := `,
		CatsFilter AS (
			SELECT link_id
			FROM global_cats_fts` + match_clause + `
		)`

	// prepend CTE
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_BASE_CTES,
		LINKS_BASE_CTES+cats_CTE,
		1)

	// append join
	const LINKS_CATS_JOIN = `
	INNER JOIN CatsFilter f ON l.id = f.link_id`
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_FROM,
		LINKS_FROM+LINKS_CATS_JOIN,
		1,
	)

	// append limit arg
	tl.Args = append(tl.Args, LINKS_PAGE_LIMIT)

	return tl
}

func (tl *TopLinks) DuringPeriod(period string) *TopLinks {
	period_clause, err := GetPeriodClause(period)
	if err != nil {
		tl.Error = err
		return tl
	}

	tl.Text = strings.Replace(
		tl.Text,
		LINKS_ORDER_BY,
		"\n"+"AND "+period_clause+LINKS_ORDER_BY,
		1,
	)

	return tl
}

func (tl *TopLinks) SortBy(order_by string) *TopLinks {

	// acceptable order_by values:
	// newest
	// rating (default)

	var updated_order string
	switch order_by {
	case "newest":
		updated_order = "submit_date DESC, like_count DESC, summary_count DESC"
	case "rating":
		updated_order = "like_count DESC, summary_count DESC, submit_date DESC"
	default:
		tl.Error = fmt.Errorf("invalid order_by value")
		return tl
	}

	updated_order_by_clause := `
	ORDER BY ` + updated_order
	
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_ORDER_BY,
		updated_order_by_clause,
		1,
	)

	return tl
}

func (tl *TopLinks) AsSignedInUser(req_user_id string) *TopLinks {

	auth_replacer := strings.NewReplacer(

		// append auth CTEs
		LINKS_BASE_CTES, LINKS_BASE_CTES+LINKS_AUTH_CTES,
		// append auth fields
		LINKS_BASE_FIELDS, LINKS_BASE_FIELDS+LINKS_AUTH_FIELDS,
		// apend auth joins
		LINKS_BASE_JOINS, LINKS_BASE_JOINS+LINKS_AUTH_JOINS,
	)

	tl.Text = auth_replacer.Replace(tl.Text)

	// prepend args
	tl.Args = append([]interface{}{req_user_id, req_user_id}, tl.Args...)

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

	// remove NSFW clause
	tl.Text = strings.Replace(
		tl.Text,
		LINKS_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// replace .DuringPeriod clause AND with WHERE
	tl.Text = strings.Replace(
		tl.Text,
		"AND submit_date",
		"WHERE submit_date",
		1,
	)

	return tl
}

func (tl *TopLinks) Page(page int) *TopLinks {
	if page == 0 {
		return tl
	}

	if page >= 1 {
		// pop limit arg and replace with limit + 1
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
	
	
	// append offset arg
	tl.Args = append(tl.Args, (page-1)*LINKS_PAGE_LIMIT)

	return tl
}
