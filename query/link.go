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

func (l *TopLinks) FromCats(cats []string) *TopLinks {
	if len(cats) == 0 || cats[0] == "" {
		l.Error = fmt.Errorf("no cats provided")
		return l
	}
	// pop limit arg
	l.Args = l.Args[:len(l.Args)-1]

	
	// build and add match arg
	EscapeCatsReservedChars(cats)
	var match_arg = cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}
	l.Args = append(l.Args, match_arg)

	// build CTE from clause
	clause := `
	WHERE global_cats MATCH ?`
	cats_cte := `,
		CatsFilter AS (
			SELECT link_id
			FROM global_cats_fts` + clause + `
		)`

	// prepend CTE
	l.Text = strings.Replace(
		l.Text,
		LINKS_BASE_CTES,
		LINKS_BASE_CTES+cats_cte,
		1)

	// append join
	const LINKS_CATS_JOIN = `
	INNER JOIN CatsFilter f ON l.id = f.link_id`
	l.Text = strings.Replace(
		l.Text,
		LINKS_FROM,
		LINKS_FROM+LINKS_CATS_JOIN,
		1,
	)

	// append limit arg
	l.Args = append(l.Args, LINKS_PAGE_LIMIT)

	return l
}

func (l *TopLinks) DuringPeriod(period string) *TopLinks {
	clause, err := GetPeriodClause(period)
	if err != nil {
		l.Error = err
		return l
	}

	l.Text = strings.Replace(
		l.Text,
		LINKS_ORDER_BY,
		"\n"+"AND "+clause+LINKS_ORDER_BY,
		1,
	)

	return l
}

func (l *TopLinks) SortBy(order_by string) *TopLinks {

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
		l.Error = fmt.Errorf("invalid order_by value")
		return l
	}

	updated_order_by_clause := `
	ORDER BY ` + updated_order
	
	l.Text = strings.Replace(
		l.Text,
		LINKS_ORDER_BY,
		updated_order_by_clause,
		1,
	)

	return l
}

func (l *TopLinks) AsSignedInUser(req_user_id string) *TopLinks {

	auth_replacer := strings.NewReplacer(

		// append auth CTEs
		LINKS_BASE_CTES, LINKS_BASE_CTES+LINKS_AUTH_CTES,
		// append auth fields
		LINKS_BASE_FIELDS, LINKS_BASE_FIELDS+LINKS_AUTH_FIELDS,
		// apend auth joins
		LINKS_BASE_JOINS, LINKS_BASE_JOINS+LINKS_AUTH_JOINS,
	)

	l.Text = auth_replacer.Replace(l.Text)

	// prepend args
	l.Args = append([]interface{}{req_user_id, req_user_id}, l.Args...)

	return l
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

func (l *TopLinks) NSFW() *TopLinks {

	// remove NSFW clause
	l.Text = strings.Replace(
		l.Text,
		LINKS_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// replace .DuringPeriod clause AND with WHERE
	l.Text = strings.Replace(
		l.Text,
		"AND submit_date",
		"WHERE submit_date",
		1,
	)

	return l
}

func (l *TopLinks) Page(page int) *TopLinks {
	if page == 0 {
		return l
	}

	if page >= 1 {
		// pop limit arg and replace with limit + 1
		l.Args = append(l.Args[:len(l.Args)-1], LINKS_PAGE_LIMIT+1)
	}

	if page == 1 {
		return l
	}

	l.Text = strings.Replace(
		l.Text,
		"LIMIT ?",
		"LIMIT ? OFFSET ?",
		1)
	
	
	// append offset arg
	l.Args = append(l.Args, (page-1)*LINKS_PAGE_LIMIT)

	return l
}
