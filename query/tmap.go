package query

import (
	"strings"

	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	mutil "github.com/julianlk522/modeep/model/util"
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
			Args: []any{
				login_name, 
				login_name, 
				login_name,
			},
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
UserStars AS (
    SELECT s.link_id
    FROM Stars s
    INNER JOIN Users u ON u.id = s.user_id
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
	OR l.id IN UserStars
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
	OR l.id IN UserStars
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

func (tnlc *TmapNSFWLinksCount) StarredOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		`(
	l.submitted_by = ?
	OR l.id IN UserStars
	OR l.id IN 
		(
		SELECT link_id
		FROM PossibleUserCats
		)
	)`,
		"l.id IN UserStars",
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
	OR l.id IN UserStars
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
	if period == "all" {
		return tnlc
	}

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
		"\nAND " + period_clause + ";",
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

	tnlc.Args = append(tnlc.Args, "%" + snippet + "%")

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) WithURLLacking(snippet string) *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		";",
		"\nAND url NOT LIKE ?;",
		1,
	)

	tnlc.Args = append(tnlc.Args, "%" + snippet + "%")

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) FromOptions(opts *model.TmapNSFWLinksCountOptions) *TmapNSFWLinksCount {
	if opts.OnlySection != "" {
		switch opts.OnlySection {
		case "submitted":
			tnlc.SubmittedOnly()
		case "starred":
			tnlc.StarredOnly()
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

	if opts.URLLacks != "" {
		tnlc.WithURLLacking(opts.URLLacks)
	}

	return tnlc
}

type TmapSubmitted struct {
	*Query
}

func NewTmapSubmitted(login_name string) *TmapSubmitted {
	return &TmapSubmitted{
		Query: &Query{
			Text: "WITH " + TMAP_BASE_CTES + "," +
				POSSIBLE_USER_CATS_CTE + "," +
				POSSIBLE_USER_SUMMARY_CTE +
				TMAP_BASE_FIELDS +
				TMAP_FROM +
				TMAP_BASE_JOINS +
				TMAP_NO_NSFW_CATS_WHERE +
				SUBMITTED_WHERE +
				TMAP_ORDER_BY_STARS,
			Args: []any{
				mutil.EARLIEST_STARRERS_LIMIT, 
				login_name, 
				login_name, 
				login_name,
			},
		},
	}
}

const SUBMITTED_WHERE = `
AND l.submitted_by = ?`

func (ts *TmapSubmitted) FromOptions(opts *model.TmapOptions) *TmapSubmitted {
	if len(opts.Cats) > 0 {
		ts.FromCats(opts.Cats)
	}

	if opts.AsSignedInUser != "" {
		ts.AsSignedInUser(opts.AsSignedInUser)
	}

	if opts.SortBy != "" {
		ts.SortBy(opts.SortBy)
	}

	if opts.IncludeNSFW {
		ts.NSFW()
	}

	if opts.Period != "" {
		ts.DuringPeriod(opts.Period)
	}

	if opts.URLContains != "" {
		ts.WithURLContaining(opts.URLContains)
	}

	if opts.URLLacks != "" {
		ts.WithURLLacking(opts.URLLacks)
	}

	return ts
}

func (ts *TmapSubmitted) FromCats(cats []string) *TmapSubmitted {
	ts.Query = FromUserOrGlobalCats(ts.Query, cats)
	return ts
}

func (ts *TmapSubmitted) AsSignedInUser(req_user_id string) *TmapSubmitted {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES + "," + TMAP_AUTH_CTE,
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS + TMAP_AUTH_FIELD,
		TMAP_BASE_JOINS, TMAP_BASE_JOINS + TMAP_AUTH_JOIN,
	)
	ts.Text = fields_replacer.Replace(ts.Text)

	new_args := make([]any, 0, len(ts.Args)+1)

	first_arg := ts.Args[0]
	trailing_args := ts.Args[1:]

	new_args = append(new_args, first_arg, req_user_id)
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

func (ts *TmapSubmitted) SortBy(metric string) *TmapSubmitted {
	if metric != "" && metric != "stars" {
		order_by_clause, ok := tmap_order_by_clauses[metric]
		if !ok {
			ts.Error = e.ErrInvalidSortByParams
		} else {
			ts.Text = strings.Replace(
				ts.Text,
				TMAP_ORDER_BY_STARS,
				order_by_clause,
				1,
			)
		}
	}

	return ts
}

func (ts *TmapSubmitted) DuringPeriod(period string) *TmapSubmitted {
	if period == "all" {
		return ts
	}
	
	period_clause, err := GetPeriodClause(period)
	if err != nil {
		ts.Error = err
		return ts
	}

	period_clause = strings.Replace(
		period_clause,
		"submit_date",
		"l.submit_date",
		1,
	)

	// string replaces should be no-op except for whichever order by clause
	// TmapSubmitted contains
	for _, order_by_clause := range tmap_order_by_clauses {
		ts.Text = strings.Replace(
			ts.Text,
			order_by_clause,
			"\nAND " + period_clause + order_by_clause,
			1,
		)
	}

	return ts
}

func (ts *TmapSubmitted) WithURLContaining(snippet string) *TmapSubmitted {
	for _, order_by_clause := range tmap_order_by_clauses {
		ts.Text = strings.Replace(
			ts.Text,
			order_by_clause,
			"\nAND " + "url LIKE ?" + order_by_clause,
			1,
		)
	} 

	ts.Args = append(ts.Args, "%" + snippet + "%")

	return ts
}

func (ts *TmapSubmitted) WithURLLacking(snippet string) *TmapSubmitted {
	for _, order_by_clause := range tmap_order_by_clauses {
		ts.Text = strings.Replace(
			ts.Text,
			order_by_clause,
			"AND url NOT LIKE ?" + order_by_clause,
			1,
		)
	} 

	ts.Args = append(ts.Args, "%" + snippet + "%")

	return ts
}

type TmapStarred struct {
	*Query
}

func NewTmapStarred(login_name string) *TmapStarred {
	q := &TmapStarred{
		Query: &Query{
			Text: "WITH " + USER_STARS_CTE + ",\n" +
				TMAP_BASE_CTES + "," +
				POSSIBLE_USER_CATS_CTE + "," +
				POSSIBLE_USER_SUMMARY_CTE +
				TMAP_BASE_FIELDS +
				TMAP_FROM +
				STARRED_JOIN +
				TMAP_BASE_JOINS +
				TMAP_NO_NSFW_CATS_WHERE +
				STARRED_WHERE +
				TMAP_ORDER_BY_STARS,
			Args: []any{
				login_name, 
				mutil.EARLIEST_STARRERS_LIMIT, 
				login_name,
				login_name, 
				login_name,
			},
		},
	}

	return q
}

const STARRED_JOIN = `
INNER JOIN UserStars us ON l.id = us.link_id`

const STARRED_WHERE = ` 
AND l.submitted_by != ?`

func (tc *TmapStarred) FromOptions(opts *model.TmapOptions) *TmapStarred {
	if len(opts.Cats) > 0 {
		tc.FromCats(opts.Cats)
	}

	if opts.AsSignedInUser != "" {
		tc.AsSignedInUser(opts.AsSignedInUser)
	}

	if opts.SortBy != "" {
		tc.SortBy(opts.SortBy)
	}

	if opts.IncludeNSFW {
		tc.NSFW()
	}

	if opts.Period != "" {
		tc.DuringPeriod(opts.Period)
	}

	if opts.URLContains != "" {
		tc.WithURLContaining(opts.URLContains)
	}

	if opts.URLLacks != "" {
		tc.WithURLLacking(opts.URLLacks)
	}

	return tc
}

func (tc *TmapStarred) FromCats(cats []string) *TmapStarred {
	tc.Query = FromUserOrGlobalCats(tc.Query, cats)
	return tc
}

func (tc *TmapStarred) AsSignedInUser(req_user_id string) *TmapStarred {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES + "," + TMAP_AUTH_CTE,
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS + TMAP_AUTH_FIELD,
		STARRED_JOIN, STARRED_JOIN + TMAP_AUTH_JOIN,
	)
	tc.Text = fields_replacer.Replace(tc.Text)

	new_args := make([]any, 0, len(tc.Args)+1)
	
	// login_name, earliest starrers limit
	first_2_args := make([]any, 2)
	copy(first_2_args, tc.Args[:2])

	trailing_args := tc.Args[2:]

	new_args = append(new_args, first_2_args...)
	new_args = append(new_args, req_user_id)
	new_args = append(new_args, trailing_args...)

	tc.Args = new_args
	return tc
}

func (tc *TmapStarred) NSFW() *TmapStarred {
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
		"AND l.submitted_by !=",
		"WHERE l.submitted_by !=",
		1,
	)
	return tc
}

func (tc *TmapStarred) SortBy(metric string) *TmapStarred {
	if metric != "" && metric != "stars" {
		order_by_clause, ok := tmap_order_by_clauses[metric]
		if !ok {
			tc.Error = e.ErrInvalidSortByParams
		} else {
			tc.Text = strings.Replace(
				tc.Text,
				TMAP_ORDER_BY_STARS,
				order_by_clause,
				1,
			)
		}
	}

	return tc
}

func (tc *TmapStarred) DuringPeriod(period string) *TmapStarred {
	if period == "all" {
		return tc
	}
	
	period_clause, err := GetPeriodClause(period)
	if err != nil {
		tc.Error = err
		return tc
	}

	period_clause = strings.Replace(
		period_clause,
		"submit_date",
		"l.submit_date",
		1,
	)

	for _, order_by_clause := range tmap_order_by_clauses {
		tc.Text = strings.Replace(
			tc.Text,
			order_by_clause,
			"\nAND " + period_clause + order_by_clause,
			1,
		)
	}

	return tc
}

func (tc *TmapStarred) WithURLContaining(snippet string) *TmapStarred {
	for _, order_by_clause := range tmap_order_by_clauses {
		tc.Text = strings.Replace(
			tc.Text,
			order_by_clause,
			"\nAND " + "url LIKE ?" + order_by_clause,
			1,
		)
	} 

	tc.Args = append(tc.Args, "%" + snippet + "%")

	return tc
}

func (tc *TmapStarred) WithURLLacking(snippet string) *TmapStarred {
	for _, order_by_clause := range tmap_order_by_clauses {
		tc.Text = strings.Replace(
			tc.Text,
			order_by_clause,
			"AND url NOT LIKE ?" + order_by_clause,
			1,
		)
	} 

	tc.Args = append(tc.Args, "%" + snippet + "%")

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
				USER_STARS_CTE +
				TAGGED_FIELDS +
				TMAP_FROM +
				TAGGED_JOINS +
				TMAP_NO_NSFW_CATS_WHERE +
				TAGGED_WHERE +
				TMAP_ORDER_BY_STARS,
			// login_name used in UserCats, PossibleUserSummary, UserStars, where
			Args: []any{
				mutil.EARLIEST_STARRERS_LIMIT, 
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
	STARRED_JOIN,
	"INNER",
	"LEFT",
	1,
)

const TAGGED_WHERE = `
AND l.submitted_by != ?
AND l.id NOT IN
	(SELECT link_id FROM UserStars)`

func (tt *TmapTagged) FromOptions(opts *model.TmapOptions) *TmapTagged {
	if len(opts.Cats) > 0 {
		tt.FromCats(opts.Cats)
	}
	
	if opts.AsSignedInUser != "" {
		tt.AsSignedInUser(opts.AsSignedInUser)
	}
	
	if opts.SortBy != "" {
		tt.SortBy(opts.SortBy)
	}
	
	if opts.IncludeNSFW {
		tt.NSFW()
	}
	
	if opts.Period != "" {
		tt.DuringPeriod(opts.Period)
	}
	
	if opts.URLContains != "" {
		tt.WithURLContaining(opts.URLContains)
	}

	if opts.URLLacks != "" {
		tt.WithURLLacking(opts.URLLacks)
	}

	return tt
}

func (tt *TmapTagged) FromCats(cats []string) *TmapTagged {
	if len(cats) == 0 || cats[0] == "" {
		return tt
	}

	// Append MATCH clause
	match_clause := `
	AND uct.user_cats MATCH ?`

	tt.Text = strings.Replace(
		tt.Text,
		TMAP_ORDER_BY_STARS,
		match_clause + TMAP_ORDER_BY_STARS,
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
		TMAP_BASE_CTES, TMAP_BASE_CTES + ","+TMAP_AUTH_CTE,
		TAGGED_FIELDS, TAGGED_FIELDS + TMAP_AUTH_FIELD,
		TAGGED_JOINS, TAGGED_JOINS + TMAP_AUTH_JOIN,
	)
	tt.Text = fields_replacer.Replace(tt.Text)

	new_args := make([]any, 0, len(tt.Args)+1)

	first_arg := tt.Args[0]
	trailing_args := tt.Args[1:]

	new_args = append(new_args, first_arg, req_user_id)
	new_args = append(new_args, trailing_args...)

	tt.Args = new_args
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
		"AND l.submitted_by !=",
		"WHERE l.submitted_by !=",
		1,
	)
	return tt
}

func (tt *TmapTagged) SortBy(metric string) *TmapTagged {
	if metric != "" && metric != "stars" {
		order_by_clause, ok := tmap_order_by_clauses[metric]
		if !ok {
			tt.Error = e.ErrInvalidSortByParams
		} else {
			tt.Text = strings.Replace(
				tt.Text,
				TMAP_ORDER_BY_STARS,
				order_by_clause,
				1,
			)
		}
	}

	return tt
}

func (tt *TmapTagged) DuringPeriod(period string) *TmapTagged {
	if period == "all" {
		return tt
	}
	
	period_clause, err := GetPeriodClause(period)
	if err != nil {
		tt.Error = err
		return tt
	}

	period_clause = strings.Replace(
		period_clause,
		"submit_date",
		"l.submit_date",
		1,
	)

for _, order_by_clause := range tmap_order_by_clauses {
		tt.Text = strings.Replace(
			tt.Text,
			order_by_clause,
			"\nAND " + period_clause + order_by_clause,
			1,
		)
	}

	return tt
}

func (tt *TmapTagged) WithURLContaining(snippet string) *TmapTagged {
	for _, order_by_clause := range tmap_order_by_clauses {
		tt.Text = strings.Replace(
			tt.Text,
			order_by_clause,
			"\nAND " + "url LIKE ?" + order_by_clause,
			1,
		)
	} 

	tt.Args = append(tt.Args, "%" + snippet + "%")

	return tt
}

func (tt *TmapTagged) WithURLLacking(snippet string) *TmapTagged {
	for _, order_by_clause := range tmap_order_by_clauses {
		tt.Text = strings.Replace(
			tt.Text,
			order_by_clause,
			"AND url NOT LIKE ?" + order_by_clause,
			1,
		)
	} 

	tt.Args = append(tt.Args, "%" + snippet + "%")

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
		PUC_WHERE + `
		AND cats MATCH ?`,
		1,
	)

	// Insert GlobalCatsFTS CTE
	q.Text = strings.Replace(
		q.Text,
		TMAP_BASE_FIELDS,
		GLOBAL_CATS_CTE + TMAP_BASE_FIELDS,
		1,
	)

	// Build MATCH arg
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}

	// Rebuild args with match_arg * 2 (once for PossibleUserCats CTE, once
	// for GlobalCatsFTS CTE)

	// TmapSubmitted args order: starrers limit, starrers limit, login_name, MATCH, login_name, MATCH, login_name
	// TmapStarred args order: login_name, starrers limit, starrers limit,  login_name, MATCH, login_name, MATCH, login_name

	// (Only TmapStarred and TmapTagged contain USER_STARS_CTE, and TmapTagged
	// does not call this method, so can check for presence of USER_STARS_CTE
	// to determine whether TmapSubmitted or TmapStarred)

	// 4th arg is login_name regardless
	login_name := q.Args[3].(string)

	// TmapStarred
	if strings.Contains(q.Text, USER_STARS_CTE) {
		q.Args = []any{
			login_name, 
			mutil.EARLIEST_STARRERS_LIMIT, 
			login_name, 
			match_arg, 
			login_name, match_arg, 
			login_name,
		}
		// TmapSubmitted
	} else {
		q.Args = []any{
			mutil.EARLIEST_STARRERS_LIMIT, 
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
		TMAP_BASE_JOINS + GLOBAL_CATS_JOIN,
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
		TMAP_ORDER_BY_STARS,
		and_clause + TMAP_ORDER_BY_STARS,
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
TimesStarred AS (
    SELECT link_id, COUNT(*) AS times_starred
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

const USER_STARS_CTE = `UserStars AS (
    SELECT s.link_id
    FROM Stars s
    INNER JOIN Users u ON u.id = s.user_id
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
    COALESCE(ts.times_starred, 0) AS times_starred,
	COALESCE(es.earliest_starrers, '') AS earliest_starrers,
	COALESCE(clc.click_count, 0) AS click_count,
    COALESCE(tc.tag_count, 0) AS tag_count,
    COALESCE(l.img_file, '') AS img_file`

const TMAP_FROM = LINKS_FROM

const TMAP_BASE_JOINS = `
LEFT JOIN PossibleUserCats puc ON l.id = puc.link_id
LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id
LEFT JOIN TimesStarred ts ON l.id = ts.link_id
LEFT JOIN EarliestStarrers es ON l.id = es.link_id
LEFT JOIN ClickCount clc ON l.id = clc.link_id
LEFT JOIN TagCount tc ON l.id = tc.link_id
LEFT JOIN SummaryCount sc ON l.id = sc.link_id`

const TMAP_NO_NSFW_CATS_WHERE = LINKS_NO_NSFW_CATS_WHERE

var tmap_order_by_clauses = map[string]string{
	"stars": TMAP_ORDER_BY_STARS,
	"newest": TMAP_ORDER_BY_NEWEST,
	"oldest": TMAP_ORDER_BY_OLDEST,
	"clicks": TMAP_ORDER_BY_CLICKS,
}

const TMAP_ORDER_BY_STARS = `
ORDER BY 
	ts.times_starred DESC, 
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, l.id DESC,
	l.submit_date DESC,
	l.id DESC;`

const TMAP_ORDER_BY_NEWEST = `
ORDER BY 
	l.submit_date DESC, 
	ts.times_starred DESC, 
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, 
	l.id DESC;`

const TMAP_ORDER_BY_OLDEST = `
ORDER BY 
	l.submit_date ASC, 
	ts.times_starred DESC, 
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, 
	l.id DESC;`

const TMAP_ORDER_BY_CLICKS = `
ORDER BY 
	clc.click_count DESC, 
	ts.times_starred DESC, 
	tc.tag_count DESC,
	sc.summary_count DESC, 
	l.id DESC;`

// Authenticated
const TMAP_AUTH_CTE = `
StarsAssigned AS (
	SELECT link_id, num_stars AS stars_assigned
	FROM Stars
	WHERE user_id = ?
	GROUP BY link_id
)`

const TMAP_AUTH_FIELD = `, 
	COALESCE(sa.stars_assigned,0) AS stars_assigned`

const TMAP_AUTH_JOIN = `
	LEFT JOIN StarsAssigned sa ON l.id = sa.link_id`
