package query

import (
	"strings"

	"github.com/julianlk522/fitm/model"
	mutil "github.com/julianlk522/fitm/model/util"
)

type TmapProfile struct {
	*Query
}

func NewTmapProfile(login_name string) *TmapProfile {
	return (&TmapProfile{
		&Query{
			Text: TMAP_PROFILE,
			Args: []any{login_name},
		},
	})
}

const TMAP_PROFILE = `SELECT 
	login_name, 
	COALESCE(pfp,'') as pfp, 
	COALESCE(about,'') as about,
	COALESCE(email,'') as email,
	created
FROM Users 
WHERE login_name = ?;`

type TmapNSFWLinksCount struct {
	*Query
}

func NewTmapNSFWLinksCount(login_name string) *TmapNSFWLinksCount {
	return &TmapNSFWLinksCount{
		&Query{
			Text: TMAP_NSFW_LINKS_COUNT,
			Args: []any{login_name, login_name, login_name},
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

func (tnlc *TmapNSFWLinksCount) SubmittedOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		`(
	l.submitted_by = ?
	OR l.id IN UserCopies
	OR l.id IN 
		(
		SELECT link_id
		FROM PossibleUserCats
		)
	)`,
		"l.submitted_by = ?",
		1,
	)

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) CopiedOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		`(
	l.submitted_by = ?
	OR l.id IN UserCopies
	OR l.id IN 
		(
		SELECT link_id
		FROM PossibleUserCats
		)
	)`,
		"l.id IN UserCopies",
		1,
	)

	tnlc.Args = tnlc.Args[:len(tnlc.Args)-1]

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) TaggedOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		`
	l.submitted_by = ?
	OR l.id IN UserCopies
	OR l.id IN 
		(
		SELECT link_id
		FROM PossibleUserCats
		)
	`,
		`
	l.submitted_by != ?
	AND l.id IN 
		(
		SELECT link_id
		FROM PossibleUserCats
		)
	`,
		1,
	)

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) FromCats(cats []string) *TmapNSFWLinksCount {
	if len(cats) == 0 || cats[0] == "" {
		return tnlc
	}

	tnlc.Text = strings.ReplaceAll(tnlc.Text, "'NSFW'", "?")

	// Build MATCH clause
	match_arg := "NSFW AND " + cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}

	// Insert match_arg arg * 2 after first login_name arg and before last 2 args
	trailing_args := make([]any, len(tnlc.Args[1:]))
	copy(trailing_args, tnlc.Args[1:])
	tnlc.Args = append(tnlc.Args[:1], match_arg, match_arg)
	tnlc.Args = append(tnlc.Args, trailing_args...)

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) DuringPeriod(period string) *TmapNSFWLinksCount {
	period_clause, err := GetPeriodClause(period)
	if err != nil {
		tnlc.Error = err
		return tnlc
	}

	period_clause = strings.Replace(
		period_clause,
		"submit_date",
		"l.submit_date",
		1,
	)

	tnlc.Text = strings.Replace(
		tnlc.Text,
		";",
		"\nAND "+period_clause + ";",
		1,
	)

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) WithURLContaining(snippet string) *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		";",
		"\nAND url LIKE ?;",
		1,
	)

	tnlc.Args = append(tnlc.Args, "%"+snippet+"%")

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) FromOptions(opts *model.TmapNSFWLinksCountOptions) *TmapNSFWLinksCount {
	if opts.OnlySection != "" {
		switch opts.OnlySection {
		case "submitted":
			tnlc.SubmittedOnly()
		case "copied":
			tnlc.CopiedOnly()
		case "tagged":
			tnlc.TaggedOnly()
		default:
			tnlc.Error = e.ErrInvalidOnlySectionParams
			return tnlc
		}
	}
	if len(opts.CatsFilter) > 0 {
		tnlc.FromCats(opts.CatsFilter)
	}
	if opts.Period != "" {
		tnlc.DuringPeriod(opts.Period)
	}
	if opts.URLContains != "" {
		tnlc.WithURLContaining(opts.URLContains)
	}

	return tnlc
}

type TmapSubmitted struct {
	*Query
}

func NewTmapSubmitted(login_name string) *TmapSubmitted {
	q := &TmapSubmitted{
		Query: &Query{
			Text: "WITH " + TMAP_BASE_CTES + "," +
				POSSIBLE_USER_CATS_CTE + "," +
				POSSIBLE_USER_SUMMARY_CTE +
				TMAP_BASE_FIELDS +
				TMAP_FROM +
				TMAP_BASE_JOINS +
				TMAP_NO_NSFW_CATS_WHERE +
				SUBMITTED_WHERE +
				TMAP_DEFAULT_ORDER_BY,
			// login_name used in PossibleUserCats, PossibleUserSummary, where
			Args: []any{
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
				login_name, 
				login_name, 
				login_name,
			},
		},
	}

	return q
}

const SUBMITTED_WHERE = `
AND l.submitted_by = ?`

func (ts *TmapSubmitted) FromCats(cats []string) *TmapSubmitted {
	ts.Query = FromUserOrGlobalCats(ts.Query, cats)
	return ts
}

func (ts *TmapSubmitted) AsSignedInUser(req_user_id string) *TmapSubmitted {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES+","+TMAP_AUTH_CTES,
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS+TMAP_AUTH_FIELDS,
		TMAP_BASE_JOINS, TMAP_BASE_JOINS+TMAP_AUTH_JOINS,
	)
	ts.Text = fields_replacer.Replace(ts.Text)

	new_args := make([]any, 0, len(ts.Args)+2)

	first_2_args := make([]any, 2)
	copy(first_2_args, ts.Args[:2])

	trailing_args := ts.Args[2:]

	new_args = append(new_args, first_2_args...)
	new_args = append(new_args, req_user_id, req_user_id)
	new_args = append(new_args, trailing_args...)

	ts.Args = new_args
	return ts
}

func (ts *TmapSubmitted) NSFW() *TmapSubmitted {
	// Remove NSFW clause
	ts.Text = strings.Replace(
		ts.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// Swap AND to WHERE in WHERE clause
	ts.Text = strings.Replace(
		ts.Text,
		"AND l.submitted_by",
		"WHERE l.submitted_by",
		1,
	)
	return ts
}

func (ts *TmapSubmitted) SortByNewest() *TmapSubmitted {
	ts.Text = strings.Replace(
		ts.Text,
		TMAP_DEFAULT_ORDER_BY,
		TMAP_ORDER_BY_NEWEST,
		1,
	)

	return ts
}

func (ts *TmapSubmitted) FromOptions(opts *model.TmapOptions) *TmapSubmitted {
	if len(opts.CatsFilter) > 0 {
		ts.FromCats(opts.CatsFilter)
	}
	if opts.AsSignedInUser != "" {
		ts.AsSignedInUser(opts.AsSignedInUser)
	}
	if opts.SortByNewest {
		ts.SortByNewest()
	}
	if opts.IncludeNSFW {
		ts.NSFW()
	}

	return ts
}

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
				TMAP_DEFAULT_ORDER_BY,
			Args: []any{
				login_name, 
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
				login_name,
				login_name, 
				login_name,
			},
		},
	}

	return q
}

const COPIED_JOIN = `
INNER JOIN UserCopies uc ON l.id = uc.link_id`

const COPIED_WHERE = ` 
AND submitted_by != ?`

func (tc *TmapCopied) FromCats(cats []string) *TmapCopied {
	tc.Query = FromUserOrGlobalCats(tc.Query, cats)
	return tc
}

func (tc *TmapCopied) AsSignedInUser(req_user_id string) *TmapCopied {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES+","+TMAP_AUTH_CTES,
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS+TMAP_AUTH_FIELDS,
		COPIED_JOIN, COPIED_JOIN+TMAP_AUTH_JOINS,
	)
	tc.Text = fields_replacer.Replace(tc.Text)

	new_args := make([]any, 0, len(tc.Args)+2)
	
	// login_name, earliest likers/copiers limit * 2
	first_3_args := make([]any, 3)
	copy(first_3_args, tc.Args[:3])

	trailing_args := tc.Args[3:]

	new_args = append(new_args, first_3_args...)
	new_args = append(new_args, req_user_id, req_user_id)
	new_args = append(new_args, trailing_args...)

	tc.Args = new_args

	return tc
}

func (tc *TmapCopied) NSFW() *TmapCopied {
	// Remove NSFW clause
	tc.Text = strings.Replace(
		tc.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// Swap AND to WHERE in WHERE clause
	tc.Text = strings.Replace(
		tc.Text,
		"AND submitted_by !=",
		"WHERE submitted_by !=",
		1,
	)
	return tc
}

func (tc *TmapCopied) SortByNewest() *TmapCopied {
	tc.Text = strings.Replace(
		tc.Text,
		TMAP_DEFAULT_ORDER_BY,
		TMAP_ORDER_BY_NEWEST,
		1,
	)

	return tc
}

func (tc *TmapCopied) FromOptions(opts *model.TmapOptions) *TmapCopied {
	if len(opts.CatsFilter) > 0 {
		tc.FromCats(opts.CatsFilter)
	}
	if opts.AsSignedInUser != "" {
		tc.AsSignedInUser(opts.AsSignedInUser)
	}
	if opts.SortByNewest {
		tc.SortByNewest()
	}
	if opts.IncludeNSFW {
		tc.NSFW()
	}

	return tc
}

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
				TMAP_DEFAULT_ORDER_BY,
			// login_name used in UserCats, PossibleUserSummary, UserCopies, where
			Args: []any{
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
				login_name, 
				login_name, 
				login_name, 
				login_name,
			},
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

func (tt *TmapTagged) FromCats(cats []string) *TmapTagged {
	if len(cats) == 0 || cats[0] == "" {
		return tt
	}

	// Append MATCH clause
	match_clause := `
	AND uct.user_cats MATCH ?`

	tt.Text = strings.Replace(
		tt.Text,
		TMAP_DEFAULT_ORDER_BY,
		match_clause+TMAP_DEFAULT_ORDER_BY,
		1,
	)

	// Append arg
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}
	tt.Args = append(tt.Args, match_arg)

	return tt
}

func (tt *TmapTagged) AsSignedInUser(req_user_id string) *TmapTagged {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES+","+TMAP_AUTH_CTES,
		TAGGED_FIELDS, TAGGED_FIELDS+TMAP_AUTH_FIELDS,
		TAGGED_JOINS, TAGGED_JOINS+TMAP_AUTH_JOINS,
	)
	tt.Text = fields_replacer.Replace(tt.Text)

	first_2_args := make([]any, 2)
	copy(first_2_args, tt.Args[:2])

	trailing_args := tt.Args[2:]

	tt.Args = append(first_2_args, req_user_id, req_user_id)
	tt.Args = append(tt.Args, trailing_args...)

	return tt
}

func (tt *TmapTagged) NSFW() *TmapTagged {
	// Remove NSFW clause
	tt.Text = strings.Replace(
		tt.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// Swap AND to WHERE in WHERE clause
	tt.Text = strings.Replace(
		tt.Text,
		"AND submitted_by !=",
		"WHERE submitted_by !=",
		1,
	)
	return tt
}

func (tt *TmapTagged) SortByNewest() *TmapTagged {
	tt.Text = strings.Replace(
		tt.Text,
		TMAP_DEFAULT_ORDER_BY,
		TMAP_ORDER_BY_NEWEST,
		1,
	)

	return tt
}

func (tt *TmapTagged) FromOptions(opts *model.TmapOptions) *TmapTagged {
	if len(opts.CatsFilter) > 0 {
		tt.FromCats(opts.CatsFilter)
	}
	if opts.AsSignedInUser != "" {
		tt.AsSignedInUser(opts.AsSignedInUser)
	}
	if opts.SortByNewest {
		tt.SortByNewest()
	}
	if opts.IncludeNSFW {
		tt.NSFW()
	}

	return tt
}

func FromUserOrGlobalCats(q *Query, cats []string) *Query {
	if len(cats) == 0 || cats[0] == "" {
		return q
	}

	// Append MATCH clause to PossibleUserCats CTE
	PUC_WHERE := "WHERE submitted_by = ?"
	q.Text = strings.Replace(
		q.Text,
		PUC_WHERE,
		PUC_WHERE+`
		AND cats MATCH ?`,
		1,
	)

	// Insert GlobalCatsFTS CTE
	q.Text = strings.Replace(
		q.Text,
		TMAP_BASE_FIELDS,
		GLOBAL_CATS_CTE+TMAP_BASE_FIELDS,
		1,
	)

	// Build MATCH arg
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}

	// Rebuild args with match_arg * 2 (once for PossibleUserCats CTE, once
	// for GlobalCatsFTS CTE)

	// TmapSubmitted args order: likers/copiers limit, likers/copiers limit, login_name, MATCH, login_name, MATCH, login_name
	// TmapCopied args order: login_name, likers/copiers limit, likers/copiers limit,  login_name, MATCH, login_name, MATCH, login_name

	// (Only TmapCopied and TmapTagged contain USER_COPIES_CTE, and TmapTagged
	// does not call this method, so can check for presence of USER_COPIES_CTE
	// to determine whether TmapSubmitted or TmapCopied)

	// 4th arg is login_name regardless
	login_name := q.Args[3].(string)

	// TmapCopied
	if strings.Contains(q.Text, USER_COPIES_CTE) {
		q.Args = []any{
			login_name, 
			mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
			mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
			login_name, 
			match_arg, 
			login_name, match_arg, 
			login_name,
		}
		// TmapSubmitted
	} else {
		q.Args = []any{
			mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
			mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT,
			login_name, 
			match_arg, 
			login_name, 
			match_arg, 
			login_name,
		}
	}

	// Insert GLOBAL_CATS_JOIN
	q.Text = strings.Replace(
		q.Text,
		TMAP_BASE_JOINS,
		TMAP_BASE_JOINS+GLOBAL_CATS_JOIN,
		1,
	)

	// Insert final AND clause
	and_clause := `
	AND (
	gc.global_cats IS NOT NULL
	OR
	puc.user_cats IS NOT NULL
)`
	q.Text = strings.Replace(
		q.Text,
		TMAP_DEFAULT_ORDER_BY,
		and_clause+TMAP_DEFAULT_ORDER_BY,
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
	COALESCE(el.earliest_likers, '') AS earliest_likers,
	COALESCE(cpc.copy_count, 0) AS copy_count,
	COALESCE(ec.earliest_copiers, '') AS earliest_copiers,
	COALESCE(clc.click_count, 0) AS click_count,
    COALESCE(tc.tag_count, 0) AS tag_count,
    COALESCE(l.img_file, '') AS img_file`

const TMAP_FROM = LINKS_FROM

const TMAP_BASE_JOINS = `
LEFT JOIN PossibleUserCats puc ON l.id = puc.link_id
LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id
LEFT JOIN LikeCount lc ON l.id = lc.link_id
LEFT JOIN EarliestLikers el ON l.id = el.link_id
LEFT JOIN CopyCount cpc ON l.id = cpc.link_id
LEFT JOIN EarliestCopiers ec ON l.id = ec.link_id
LEFT JOIN ClickCount clc ON l.id = clc.link_id
LEFT JOIN TagCount tc ON l.id = tc.link_id
LEFT JOIN SummaryCount sc ON l.id = sc.link_id`

const TMAP_NO_NSFW_CATS_WHERE = LINKS_NO_NSFW_CATS_WHERE

const TMAP_DEFAULT_ORDER_BY = `
ORDER BY 
	lc.like_count DESC, 
	cpc.copy_count DESC,
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, l.id DESC,
	l.submit_date DESC,
	l.id DESC;`

const TMAP_ORDER_BY_NEWEST = `
ORDER BY 
	l.submit_date DESC, 
	lc.like_count DESC, 
	cpc.copy_count DESC,
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, 
	l.id DESC;`

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
