package query

import (
	"strings"
)

// PROFILE
type TmapProfile struct {
	*Query
}
func NewTmapProfile(login_name string) *TmapProfile {
	return (&TmapProfile{
		&Query{
			Text: TMAP_PROFILE,
			Args: []interface{}{login_name},
		},
	})
}

const TMAP_PROFILE = `SELECT 
	login_name, 
	COALESCE(about,'') as about, 
	COALESCE(pfp,'') as pfp, 
	created
FROM Users 
WHERE login_name = ?;`

// NSFW LINKS COUNT
type TmapNSFWLinksCount struct {
	*Query
}

func NewTmapNSFWLinksCount(login_name string) *TmapNSFWLinksCount {
	return &TmapNSFWLinksCount{
		&Query{
			Text: TMAP_NSFW_LINKS_COUNT,
			Args: []interface{}{login_name, login_name, login_name},
		},
	}
}

const TMAP_NSFW_LINKS_COUNT = `WITH PossibleUserCats AS (
    SELECT 
		link_id, 
		cats AS user_cats,
		(cats IS NOT NULL) AS cats_from_user
    FROM user_cats_fts
    WHERE submitted_by = ?
	AND cats MATCH 'NSFW'
),
GlobalCatsFTS AS (
	SELECT
		link_id,
		global_cats
	FROM global_cats_fts
	WHERE global_cats MATCH 'NSFW'
),
UserCopies AS (
    SELECT lc.link_id
    FROM "Link Copies" lc
    INNER JOIN Users u ON u.id = lc.user_id
    WHERE u.login_name = ?
)
SELECT count(*) as NSFW_link_count
FROM Links l
LEFT JOIN PossibleUserCats puc ON l.id = puc.link_id
LEFT JOIN GlobalCatsFTS gc ON l.id = gc.link_id
WHERE 
	(
	gc.global_cats IS NOT NULL
	OR
	puc.user_cats IS NOT NULL
	)
AND (
	l.submitted_by = ?
	OR l.id IN UserCopies
	OR l.id IN 
		(
		SELECT link_id
		FROM PossibleUserCats
		)
	);`

func (lc *TmapNSFWLinksCount) FromCats(cats []string) *TmapNSFWLinksCount {
	if len(cats) == 0 || cats[0] == "" {
		return lc
	}

	lc.Text = strings.ReplaceAll(lc.Text, "'NSFW'", "?")

	// build MATCH clause
	cat_match := "NSFW AND " + cats[0]
	for i := 1; i < len(cats); i++ {
		cat_match += " AND " + cats[i]
	}

	// insert cat_match arg * 2 after first login_name arg and before last 2
	// copy trailing args to re-append after insert
	trailing_args := make([]interface{}, len(lc.Args[1:]))
	copy(trailing_args, lc.Args[1:])

	// insert
	lc.Args = append(lc.Args[:1], cat_match, cat_match)
	lc.Args = append(lc.Args, trailing_args...)
	
	return lc
}

// LINKS
// Submitted links (global cats replaced with user-assigned if user's tag remains)
type TmapSubmitted struct {
	*Query
}

func NewTmapSubmitted(login_name string) *TmapSubmitted {
	q := &TmapSubmitted{
		Query: &Query{
			Text: 
				"WITH " + TMAP_BASE_CTES + "," +
				POSSIBLE_USER_CATS_CTE + "," +
				POSSIBLE_USER_SUMMARY_CTE +
				TMAP_BASE_FIELDS +
				TMAP_FROM +
				TMAP_BASE_JOINS +
				TMAP_NO_NSFW_CATS_WHERE +
				SUBMITTED_WHERE +
				TMAP_ORDER_BY,
			// login_name used in PossibleUserCats, PossibleUserSummary, where
			Args: []interface{}{login_name, login_name, login_name},
		},
	}

	return q
}

const SUBMITTED_WHERE = `
AND l.submitted_by = ?`

func (q *TmapSubmitted) FromCats(cats []string) *TmapSubmitted {
	q.Query = FromUserOrGlobalCats(q.Query, cats)
	return q
}

func (q *TmapSubmitted) AsSignedInUser(req_user_id string) *TmapSubmitted {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES+","+TMAP_AUTH_CTES,
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS+TMAP_AUTH_FIELDS,
		TMAP_BASE_JOINS, TMAP_BASE_JOINS+TMAP_AUTH_JOINS,
	)
	q.Text = fields_replacer.Replace(q.Text)

	// prepend req_user_id arg * 2
	q.Args = append([]interface{}{req_user_id, req_user_id}, q.Args...)

	return q
}

func (q *TmapSubmitted) NSFW() *TmapSubmitted {

	// remove NSFW clause
	q.Text = strings.Replace(
		q.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// swap AND to WHERE in WHERE clause
	q.Text = strings.Replace(
		q.Text,
		"AND l.submitted_by",
		"WHERE l.submitted_by",
		1,
	)
	return q
}

// Copied links submitted by other users (global cats replaced with user-assigned if user has tagged)
type TmapCopied struct {
	*Query
}

func NewTmapCopied(login_name string) *TmapCopied {
	q := &TmapCopied{
		Query: &Query{
			Text: "WITH " + USER_COPIES_CTE + ",\n" +
				TMAP_BASE_CTES + "," +
				POSSIBLE_USER_CATS_CTE + "," +
				POSSIBLE_USER_SUMMARY_CTE +
				TMAP_BASE_FIELDS +
				TMAP_FROM +
				COPIED_JOIN +
				TMAP_BASE_JOINS +
				TMAP_NO_NSFW_CATS_WHERE +
				COPIED_WHERE +
				TMAP_ORDER_BY,
			// login_name used in UserCopies, PossibleUserCats, 
			// PossibleUserSummary, where
			Args: []interface{}{login_name, login_name, login_name, login_name},
		},
	}

	return q
}

const COPIED_JOIN = `
INNER JOIN UserCopies uc ON l.id = uc.link_id`

const COPIED_WHERE = ` 
AND submitted_by != ?`

func (q *TmapCopied) FromCats(cats []string) *TmapCopied {
	q.Query = FromUserOrGlobalCats(q.Query, cats)
	return q
}

func (q *TmapCopied) AsSignedInUser(req_user_id string) *TmapCopied {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES+","+TMAP_AUTH_CTES,
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS+TMAP_AUTH_FIELDS,
		COPIED_JOIN, COPIED_JOIN+TMAP_AUTH_JOINS,
	)
	q.Text = fields_replacer.Replace(q.Text)

	// prepend req_user_id arg * 2
	q.Args = append([]interface{}{req_user_id, req_user_id}, q.Args...)

	return q
}

func (q *TmapCopied) NSFW() *TmapCopied {

	// remove NSFW clause
	q.Text = strings.Replace(
		q.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// swap AND to WHERE in WHERE clause
	q.Text = strings.Replace(
		q.Text,
		"AND submitted_by !=",
		"WHERE submitted_by !=",
		1,
	)
	return q
}

// Tagged links submitted by other users (global cats replaced with user-assigned)
type TmapTagged struct {
	*Query
}

func NewTmapTagged(login_name string) *TmapTagged {
	q := &TmapTagged{
		Query: &Query{
			Text: "WITH " + TMAP_BASE_CTES + ",\n" +
				USER_CATS_CTE + "," +
				POSSIBLE_USER_SUMMARY_CTE + ",\n" +
				USER_COPIES_CTE +
				TAGGED_FIELDS +
				TMAP_FROM +
				TAGGED_JOINS +
				TMAP_NO_NSFW_CATS_WHERE +
				TAGGED_WHERE +
				TMAP_ORDER_BY,
			// login_name used in UserCats, PossibleUserSummary, UserCopies, where
			Args: []interface{}{login_name, login_name, login_name, login_name},
		},
	}

	q.Text = strings.ReplaceAll(q.Text, "LOGIN_NAME", login_name)
	return q
}

var TAGGED_FIELDS = strings.Replace(
	strings.Replace(
		TMAP_BASE_FIELDS,
		"COALESCE(puc.user_cats, l.global_cats) AS cats",
		"uct.user_cats",
		1,
	),
	`COALESCE(puc.cats_from_user,0) AS cats_from_user`,
	"1 AS cats_from_user",
	1,
)

var TAGGED_JOINS = strings.Replace(
	TMAP_BASE_JOINS,
	"LEFT JOIN PossibleUserCats puc ON l.id = puc.link_id",
	"INNER JOIN UserCats uct ON l.id = uct.link_id",
	1,
) + strings.Replace(
	COPIED_JOIN,
	"INNER",
	"LEFT",
	1,
)

const TAGGED_WHERE = `
AND submitted_by != ?
AND l.id NOT IN
	(SELECT link_id FROM UserCopies)`

func (q *TmapTagged) FromCats(cats []string) *TmapTagged {
	if len(cats) == 0 || cats[0] == "" {
		return q
	}

	// append clause
	cat_clause := `
	AND uct.user_cats MATCH ?`

	q.Text = strings.Replace(
		q.Text,
		TMAP_ORDER_BY,
		cat_clause+TMAP_ORDER_BY,
		1,
	)

	// append arg
	cat_match := cats[0]
	for i := 1; i < len(cats); i++ {
		cat_match += " AND " + cats[i]
	}
	q.Args = append(q.Args, cat_match)

	return q
}

func (q *TmapTagged) AsSignedInUser(req_user_id string) *TmapTagged {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES+","+TMAP_AUTH_CTES,
		TAGGED_FIELDS, TAGGED_FIELDS+TMAP_AUTH_FIELDS,
		TAGGED_JOINS, TAGGED_JOINS+TMAP_AUTH_JOINS,
	)
	q.Text = fields_replacer.Replace(q.Text)

	// prepend req_user_id arg * 2
	q.Args = append([]interface{}{req_user_id, req_user_id}, q.Args...)

	return q
}

func (q *TmapTagged) NSFW() *TmapTagged {

	// remove NSFW clause
	q.Text = strings.Replace(
		q.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// swap AND to WHERE in WHERE clause
	q.Text = strings.Replace(
		q.Text,
		"AND submitted_by !=",
		"WHERE submitted_by !=",
		1,
	)
	return q
}

func FromUserOrGlobalCats(q *Query, cats []string) *Query {
	if len(cats) == 0 || cats[0] == "" {
		return q
	}

	// append MATCH clause to PossibleUserCats CTE
	PUC_WHERE := "WHERE submitted_by = ?"
	q.Text = strings.Replace(
		q.Text,
		PUC_WHERE,
		PUC_WHERE+`
		AND cats MATCH ?`,
		1,
	)

	// insert GlobalCatsFTS CTE
	q.Text = strings.Replace(
		q.Text,
		TMAP_BASE_FIELDS,
		GLOBAL_CATS_CTE+TMAP_BASE_FIELDS,
		1,
	)

	// build MATCH arg
	cat_match := cats[0]
	for i := 1; i < len(cats); i++ {
		cat_match += " AND " + cats[i]
	}

	// rebuild args with cat_match * 2 (once for PossibleUserCats CTE, once
	// for GlobalCatsFTS CTE) 
	// order for TmapSubmitted: login_name, MATCH, login_name, MATCH, login_name 
	// order for TmapCopied: login_name, login_name, MATCH, login_name, 
	// MATCH, login_name

	// (only TmapCopied and TmapTagged contain USER_COPIES_CTE, and TmapTagged
	// does not call this method, so can check for presence of USER_COPIES_CTE
	// to determine whether TmapSubmitted or TmapCopied)

	// get login_name from first arg
	login_name := q.Args[0].(string)

	// TmapCopied
	if strings.Contains(q.Text, USER_COPIES_CTE) {
		q.Args = []interface{}{login_name, login_name, cat_match, login_name, cat_match, login_name}
	// TmapSubmitted
	} else {
		q.Args = []interface{}{login_name, cat_match, login_name, cat_match, login_name}
	}

	// insert GLOBAL_CATS_JOIN
	q.Text = strings.Replace(
		q.Text,
		TMAP_BASE_JOINS,
		TMAP_BASE_JOINS+GLOBAL_CATS_JOIN,
		1,
	)

	// insert final AND clause
	and_clause := `
	AND (
	gc.global_cats IS NOT NULL
	OR
	puc.user_cats IS NOT NULL
)`
	q.Text = strings.Replace(
		q.Text,
		TMAP_ORDER_BY,
		and_clause+TMAP_ORDER_BY,
		1,
	)

	return q
}

const GLOBAL_CATS_CTE = `,
	GlobalCatsFTS AS (
		SELECT
			link_id,
			global_cats
		FROM global_cats_fts
		WHERE global_cats MATCH ?
	)`

const GLOBAL_CATS_JOIN = `
LEFT JOIN GlobalCatsFTS gc ON l.id = gc.link_id`

const USER_CATS_CTE = `UserCats AS (
    SELECT link_id, cats as user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
)`

// Base
const TMAP_BASE_CTES = `SummaryCount AS (
    SELECT link_id, COUNT(*) AS summary_count
    FROM Summaries
    GROUP BY link_id
),
LikeCount AS (
    SELECT link_id, COUNT(*) AS like_count
    FROM "Link Likes"
    GROUP BY link_id
),
TagCount AS (
    SELECT link_id, COUNT(*) AS tag_count
    FROM Tags
    GROUP BY link_id
)`

const POSSIBLE_USER_CATS_CTE = `
PossibleUserCats AS (
    SELECT 
		link_id, 
		cats AS user_cats,
		(cats IS NOT NULL) AS cats_from_user
    FROM user_cats_fts
    WHERE submitted_by = ?
)`

const POSSIBLE_USER_SUMMARY_CTE = `
PossibleUserSummary AS (
    SELECT
        link_id, 
		text as user_summary
    FROM Summaries
    INNER JOIN Users u ON u.id = submitted_by
	WHERE u.login_name = ?
)`

const USER_COPIES_CTE = `UserCopies AS (
    SELECT lc.link_id
    FROM "Link Copies" lc
    INNER JOIN Users u ON u.id = lc.user_id
    WHERE u.login_name = ?
)`

const TMAP_BASE_FIELDS = `
SELECT 
	l.id AS link_id,
    l.url,
    l.submitted_by AS login_name,
    l.submit_date,
    COALESCE(puc.user_cats, l.global_cats) AS cats,
    COALESCE(puc.cats_from_user,0) AS cats_from_user,
    COALESCE(pus.user_summary, l.global_summary, '') AS summary,
    COALESCE(sc.summary_count, 0) AS summary_count,
    COALESCE(lc.like_count, 0) AS like_count,
    COALESCE(tc.tag_count, 0) AS tag_count,
    COALESCE(l.img_url, '') AS img_url`

const TMAP_FROM = LINKS_FROM

const TMAP_BASE_JOINS = `
LEFT JOIN PossibleUserCats puc ON l.id = puc.link_id
LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id
LEFT JOIN TagCount tc ON l.id = tc.link_id
LEFT JOIN LikeCount lc ON l.id = lc.link_id
LEFT JOIN SummaryCount sc ON l.id = sc.link_id`

const TMAP_NO_NSFW_CATS_WHERE = LINKS_NO_NSFW_CATS_WHERE

const TMAP_ORDER_BY = `
ORDER BY lc.like_count DESC, sc.summary_count DESC, l.id DESC;`

// Authenticated
const TMAP_AUTH_CTES = `
IsLiked AS (
	SELECT link_id, COUNT(*) AS is_liked
	FROM "Link Likes"
	WHERE user_id = ?
	GROUP BY id
),
IsCopied AS (
	SELECT link_id, COUNT(*) AS is_copied
	FROM "Link Copies"
	WHERE user_id = ?
	GROUP BY id
)`

const TMAP_AUTH_FIELDS = `, 
	COALESCE(is_liked,0) as is_liked, 
	COALESCE(is_copied,0) as is_copied`

const TMAP_AUTH_JOINS = `
	LEFT JOIN IsLiked il ON l.id = il.link_id
	LEFT JOIN IsCopied ic ON l.id = ic.link_id`