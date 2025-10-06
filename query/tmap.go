package query

import (
	"strings"

	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	mutil "github.com/julianlk522/modeep/model/util"
)

// TREASURE MAP PROFILE
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

// NSFW LINKS COUNT
type TmapNSFWLinksCount struct {
	*Query
}

func NewTmapNSFWLinksCount(login_name string) *TmapNSFWLinksCount {
	return &TmapNSFWLinksCount{
		&Query{
			Text: "WITH " + NSFW_CATS_CTES + "\n" +
			USER_STARS_CTE + 
			`SELECT count(*) as NSFW_link_count
				FROM Links l` + 
				NSFW_JOINS + `
				WHERE 
					(gc.global_cats IS NOT NULL
					OR
					puc.user_cats IS NOT NULL)` +
				TMAP_NSFW_LINKS_FINAL_AND + ";",
			Args: []any{
				login_name, 
				login_name, 
				login_name,
			},
		},
	}
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

	if opts.SummaryContains != "" {
		tnlc.WithSummaryContaining(opts.SummaryContains)
	}

	if opts.URLContains != "" {
		tnlc.WithURLContaining(opts.URLContains)
	}

	if opts.URLLacks != "" {
		tnlc.WithURLLacking(opts.URLLacks)
	}

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) SubmittedOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		TMAP_NSFW_LINKS_FINAL_AND,
		"\nAND l.submitted_by = ?",
		1,
	)

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) StarredOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		TMAP_NSFW_LINKS_FINAL_AND,
		"\nAND l.id IN (SELECT link_id FROM UserStars)",
		1,
	)

	tnlc.Args = tnlc.Args[:len(tnlc.Args)-1]

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) TaggedOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		TMAP_NSFW_LINKS_FINAL_AND,
		TAGGED_AND,
		1,
	)

	return tnlc
}

const TMAP_NSFW_LINKS_FINAL_AND = `
AND (
	l.submitted_by = ?
	OR l.id IN (SELECT link_id FROM UserStars)
	OR l.id IN 
		(
		SELECT link_id
		FROM PossibleUserCats_NSFW
		)
	)`

func (tnlc *TmapNSFWLinksCount) FromCats(cats []string) *TmapNSFWLinksCount {
	if len(cats) == 0 || cats[0] == "" {
		return tnlc
	}

	// Insert CTEs
	cat_filter_ctes_no_cats_from_user := strings.Replace(
		CAT_FILTER_CTES,
		`,
		(cats IS NOT NULL) AS cats_from_user`,
		"",
		1,
	)
	tnlc.Text = strings.Replace(
		tnlc.Text,
		"UserStars AS",
		cat_filter_ctes_no_cats_from_user + ",\n" + "UserStars AS",
		1,
	)

	// Insert joins
	tnlc.Text = strings.Replace(
		tnlc.Text,
		"LEFT JOIN GlobalCatsFTS_NSFW gc ON l.id = gc.link_id",
		"LEFT JOIN GlobalCatsFTS_NSFW gc ON l.id = gc.link_id" + CAT_FILTER_JOINS,
		1,
	)

	// add necessary cats condition
	tnlc.Text = strings.Replace(
		tnlc.Text,
		"puc.user_cats IS NOT NULL)",
		`puc.user_cats IS NOT NULL)
		AND (gco.global_cats IS NOT NULL OR puco.user_cats IS NOT NULL)`,
		1,
	)

	// Build MATCH clause
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}

	// Insert args
	// since other methods push new args to end of slice,
	// better to insert these from the start (always after the first 1,
	// which is login_name)
	login_name := tnlc.Args[0]
	
	trailing_args := make([]any, len(tnlc.Args[1:]))
	copy(trailing_args, tnlc.Args[1:])
	
	// args to add: login_name, match arg, match arg (again)
	tnlc.Args = append(tnlc.Args[:1], login_name, match_arg, match_arg)
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

func (tnlc *TmapNSFWLinksCount) WithSummaryContaining(snippet string) *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		POSSIBLE_USER_CATS_CTE_NO_CATS_FROM_USER,
		POSSIBLE_USER_CATS_CTE_NO_CATS_FROM_USER + "," + POSSIBLE_USER_SUMMARY_CTE,
		1,
	)
	tnlc.Text = strings.Replace(
		tnlc.Text,
		"LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id",
		"LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id" + "\n" + "LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id",
		1,
	)
	tnlc.Text = strings.Replace(
		tnlc.Text,
		";",
		"\nAND global_summary LIKE ?;",
		1,
	)

	tnlc.Args = append(tnlc.Args, "%" + snippet + "%")

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

// LINKS
type TmapSubmitted struct {
	*Query
}

func NewTmapSubmitted(login_name string) *TmapSubmitted {
	return &TmapSubmitted{
		Query: &Query{
			Text: "WITH " + TMAP_BASE_CTES + "\n" +
				NSFW_CATS_CTES +
				POSSIBLE_USER_CATS_CTE + "," +
				POSSIBLE_USER_SUMMARY_CTE +
				TMAP_BASE_FIELDS +
				TMAP_FROM +
				TMAP_BASE_JOINS +
				NSFW_JOINS + "\n" +
				TMAP_NO_NSFW_CATS_WHERE +
				SUBMITTED_AND +
				TMAP_ORDER_BY_TIMES_STARRED,
			Args: []any{
				mutil.EARLIEST_STARRERS_LIMIT, 
				login_name, 
				login_name, 
				login_name,
				login_name,
				login_name,
			},
		},
	}
}

const SUBMITTED_AND = `
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

	if opts.SummaryContains != "" {
		ts.WithSummaryContaining(opts.SummaryContains)
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
	// add CTEs and joins
	ts.Text = strings.Replace(
		ts.Text,
		POSSIBLE_USER_CATS_CTE,
		CAT_FILTER_CTES,
		1,
	)
	ts.Text = strings.Replace(
		ts.Text,
		"LEFT JOIN PossibleUserCats_Other puco ON l.id = puco.link_id",
		CAT_FILTER_JOINS,
		1,
	)
	// add necessary cats condition
	ts.Text = strings.Replace(
		ts.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		TMAP_NO_NSFW_CATS_WHERE + `
		AND (gco.global_cats IS NOT NULL OR puco.user_cats IS NOT NULL)`,
		1,
	)

	// build match arg
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}

	// insert args	
	// old: [{earliest_starrers_limit}, login_name x 5]
	// new: [{earliest_starrers_limit}, login_name x 2, 
	// match_arg x 2, login_name x 3]
	first_3_args := make([]any, 3)
	copy(first_3_args, ts.Args[:3])

	new_args := make([]any, 0, len(ts.Args) + 3)
	new_args = append(new_args, first_3_args...)
	new_args = append(new_args, match_arg, match_arg)
	new_args = append(new_args, ts.Args[3:]...)

	ts.Args = new_args
	return ts
}

func (ts *TmapSubmitted) AsSignedInUser(req_user_id string) *TmapSubmitted {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES + TMAP_AUTH_CTE + ",",
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS + TMAP_AUTH_FIELD,
		TMAP_BASE_JOINS, TMAP_BASE_JOINS + TMAP_AUTH_JOIN,
		// in case TMAP_BASE_JOINS are modified by .FromCats
		CAT_FILTER_JOINS, CAT_FILTER_JOINS + TMAP_AUTH_JOIN,
	)
	ts.Text = fields_replacer.Replace(ts.Text)

	// Insert args after first index
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
	// swap following AND to WHERE
	if strings.Contains(
		ts.Text,
		"AND (gco.global_cats IS NOT NULL OR puco.user_cats IS NOT NULL)",
	) {
		ts.Text = strings.Replace(
			ts.Text,
			"AND (gco.global_cats",
			"WHERE (gco.global_cats",
			1,
		)
	} else {
		ts.Text = strings.Replace(
			ts.Text,
			SUBMITTED_AND,
			`
WHERE l.submitted_by = ?`,
			1,
		)
	}
	
	// Remove login_name arg (it should always be last if this is called 
	// before .WithBlahBlah methods, which it should always be per the order
	// of TmapSubmitted.FromOptions())
	ts.Args = ts.Args[:len(ts.Args) - 1]

	return ts
}

func (ts *TmapSubmitted) SortBy(metric string) *TmapSubmitted {
	if metric != "" && metric != "times_starred" {
		order_by_clause, ok := tmap_order_by_clauses[metric]
		if !ok {
			ts.Error = e.ErrInvalidSortByParams
		} else {
			ts.Text = strings.Replace(
				ts.Text,
				TMAP_ORDER_BY_TIMES_STARRED,
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

func (ts *TmapSubmitted) WithSummaryContaining(snippet string) *TmapSubmitted {
	for _, order_by_clause := range tmap_order_by_clauses {
		ts.Text = strings.Replace(
			ts.Text,
			order_by_clause,
			"\nAND summary LIKE ?" + order_by_clause,
			1,
		)
	} 

	ts.Args = append(ts.Args, "%" + snippet + "%")
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
			Text: "WITH " + TMAP_BASE_CTES + "\n" +
				NSFW_CATS_CTES + "\n" +
				USER_STARS_CTE + "," +
				POSSIBLE_USER_CATS_CTE + ",\n" +
				POSSIBLE_USER_SUMMARY_CTE +
				TMAP_BASE_FIELDS +
				TMAP_FROM +
				STARRED_JOIN +
				TMAP_BASE_JOINS +
				NSFW_JOINS + "\n" +
				TMAP_NO_NSFW_CATS_WHERE +
				STARRED_AND +
				TMAP_ORDER_BY_TIMES_STARRED,
			Args: []any{
				mutil.EARLIEST_STARRERS_LIMIT, 
				login_name, 
				login_name,
				login_name,
				login_name, 
				login_name,
				login_name,
			},
		},
	}

	return q
}

const STARRED_AND = ` 
AND l.submitted_by != ?`

func (ts *TmapStarred) FromOptions(opts *model.TmapOptions) *TmapStarred {
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

	if opts.SummaryContains != "" {
		ts.WithSummaryContaining(opts.SummaryContains)
	}

	if opts.URLContains != "" {
		ts.WithURLContaining(opts.URLContains)
	}

	if opts.URLLacks != "" {
		ts.WithURLLacking(opts.URLLacks)
	}

	return ts
}

func (ts *TmapStarred) FromCats(cats []string) *TmapStarred {
	// add CTEs and joins
	ts.Text = strings.Replace(
		ts.Text,
		POSSIBLE_USER_CATS_CTE,
		CAT_FILTER_CTES,
		1,
	)
	ts.Text = strings.Replace(
		ts.Text,
		"LEFT JOIN PossibleUserCats_Other puco ON l.id = puco.link_id",
		CAT_FILTER_JOINS,
		1,
	)
	// add necessary cats condition
	ts.Text = strings.Replace(
		ts.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		TMAP_NO_NSFW_CATS_WHERE + `
		AND (gco.global_cats IS NOT NULL OR puco.user_cats IS NOT NULL)`,
		1,
	)

	// build match arg
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}

	// insert args	
	// old: [{earliest_starrers_limit}, login_name x 6]
	// new: [{earliest_starrers_limit}, login_name x 3, 
	// match_arg x 2, login_name x 3]
	// so insert before last 3
	last_3_args := ts.Args[len(ts.Args) - 3:]
	new_args := append([]any{}, ts.Args[:len(ts.Args) - 3]...)
	new_args = append(new_args, match_arg, match_arg)
	new_args = append(new_args, last_3_args...)
	ts.Args = new_args

	return ts
}

func (ts *TmapStarred) AsSignedInUser(req_user_id string) *TmapStarred {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES + TMAP_AUTH_CTE + ",",
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS + TMAP_AUTH_FIELD,
		TMAP_BASE_JOINS, TMAP_BASE_JOINS + TMAP_AUTH_JOIN,
		// in case TMAP_BASE_JOINS are modified by .FromCats
		CAT_FILTER_JOINS, CAT_FILTER_JOINS + TMAP_AUTH_JOIN,
	)
	ts.Text = fields_replacer.Replace(ts.Text)

	// insert args:
	// old: [[{earliest_starrers_limit} login_name x 3, 
	// match_arg x 2, login_name x 3]
	// new: {earliest_starrers_limit}, req_user_id, login_name x 3,
	// match_arg x 2, login_name x 3] 
	// so insert req_user_id after earliest_starrers_limit
	first_arg := ts.Args[0]
	trailing_args := ts.Args[1:]

	new_args := make([]any, 0, len(ts.Args) + 1)
	new_args = append(new_args, first_arg, req_user_id)
	new_args = append(new_args, trailing_args...)

	ts.Args = new_args
	return ts
}

func (ts *TmapStarred) NSFW() *TmapStarred {
	// Remove NSFW clause
	ts.Text = strings.Replace(
		ts.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// Swap following condition keyword
	if strings.Contains(
		ts.Text,
		"AND (gco.global_cats IS NOT NULL OR puco.user_cats IS NOT NULL)",
	) {
		ts.Text = strings.Replace(
			ts.Text,
			"AND (gco.global_cats",
			"WHERE (gco.global_cats",
			1,
		)
	} else {
		ts.Text = strings.Replace(
			ts.Text,
			STARRED_AND,
			` 
WHERE l.submitted_by != ?`,
			1,
		)
	}

	// Remove login_name arg used in TMAP_NO_NSFW_CATS_WHERE
	ts.Args = ts.Args[:len(ts.Args) - 1]
	
	return ts
}

func (ts *TmapStarred) SortBy(metric string) *TmapStarred {
	if metric != "" && metric != "times_starred" {
		order_by_clause, ok := tmap_order_by_clauses[metric]
		if !ok {
			ts.Error = e.ErrInvalidSortByParams
		} else {
			ts.Text = strings.Replace(
				ts.Text,
				TMAP_ORDER_BY_TIMES_STARRED,
				order_by_clause,
				1,
			)
		}
	}

	return ts
}

func (ts *TmapStarred) DuringPeriod(period string) *TmapStarred {
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

func (ts *TmapStarred) WithSummaryContaining(snippet string) *TmapStarred {
	for _, order_by_clause := range tmap_order_by_clauses {
		ts.Text = strings.Replace(
			ts.Text,
			order_by_clause,
			"\nAND summary LIKE ?" + order_by_clause,
			1,
		)
	} 

	ts.Args = append(ts.Args, "%" + snippet + "%")
	return ts
}

func (ts *TmapStarred) WithURLContaining(snippet string) *TmapStarred {
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

func (ts *TmapStarred) WithURLLacking(snippet string) *TmapStarred {
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

type TmapTagged struct {
	*Query
}

func NewTmapTagged(login_name string) *TmapTagged {
	q := &TmapTagged{
		Query: &Query{
			Text: "WITH " +
				TMAP_BASE_CTES + "\n" +
				NSFW_CATS_CTES + "\n" +
				USER_CATS_CTE + ",\n" +
				USER_STARS_CTE + "," +
				POSSIBLE_USER_SUMMARY_CTE + "\n" +
				TAGGED_FIELDS +
				TMAP_FROM +
				TAGGED_JOINS +
				NSFW_JOINS + "\n" +
				TMAP_NO_NSFW_CATS_WHERE +
				TAGGED_AND +
				TMAP_ORDER_BY_TIMES_STARRED,
			Args: []any{
				mutil.EARLIEST_STARRERS_LIMIT, 
				login_name, 
				login_name, 
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
		"COALESCE(puco.user_cats, l.global_cats) AS cats",
		"uct.user_cats",
		1,
	),
	`COALESCE(puco.cats_from_user,0) AS cats_from_user`,
	"1 AS cats_from_user",
	1,
)

var TAGGED_JOINS = strings.Replace(
	TMAP_BASE_JOINS,
	"LEFT JOIN PossibleUserCats_Other puco ON l.id = puco.link_id",
	"INNER JOIN UserCats uct ON l.id = uct.link_id",
	1,
) + strings.Replace(
	STARRED_JOIN,
	"INNER",
	"LEFT",
	1,
)

const TAGGED_AND = `
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

	if opts.SummaryContains != "" {
		tt.WithSummaryContaining(opts.SummaryContains)
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

	// .SortBy hasn't been called yet - can safely assume
	// it will have this one
	tt.Text = strings.Replace(
		tt.Text,
		TMAP_ORDER_BY_TIMES_STARRED,
		match_clause + TMAP_ORDER_BY_TIMES_STARRED,
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
		TMAP_BASE_CTES, TMAP_BASE_CTES + TMAP_AUTH_CTE + ",",
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

	// swap following AND to WHERE
	if strings.Contains(
		tt.Text,
		"AND (gco.global_cats IS NOT NULL OR puco.user_cats IS NOT NULL)",
	) {
		tt.Text = strings.Replace(
			tt.Text,
			"AND (gco.global_cats",
			"WHERE (gco.global_cats",
			1,
		)
	} else {
		tt.Text = strings.Replace(
			tt.Text,
			TAGGED_AND,
			`
WHERE l.submitted_by != ?
AND l.id NOT IN
	(SELECT link_id FROM UserStars)`,
			1,
		)
	}
	
	// Remove login_name arg used in TMAP_NO_NSFW_CATS_WHERE
	tt.Args = tt.Args[:len(tt.Args) - 1]

	return tt
}

func (tt *TmapTagged) SortBy(metric string) *TmapTagged {
	if metric != "" && metric != "times_starred" {
		order_by_clause, ok := tmap_order_by_clauses[metric]
		if !ok {
			tt.Error = e.ErrInvalidSortByParams
		} else {
			tt.Text = strings.Replace(
				tt.Text,
				TMAP_ORDER_BY_TIMES_STARRED,
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


func (tt *TmapTagged) WithSummaryContaining(snippet string) *TmapTagged {
	for _, order_by_clause := range tmap_order_by_clauses {
		tt.Text = strings.Replace(
			tt.Text,
			order_by_clause,
			"\nAND summary LIKE ?" + order_by_clause,
			1,
		)
	} 

	tt.Args = append(tt.Args, "%" + snippet + "%")

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
	
// Treasure Map Base
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
),`

const NSFW_CATS_CTES = `PossibleUserCats_NSFW AS (
    SELECT 
		link_id, 
		cats AS user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
	AND cats MATCH 'NSFW'
),
GlobalCatsFTS_NSFW AS (
	SELECT
		link_id,
		global_cats
	FROM global_cats_fts
	WHERE global_cats MATCH 'NSFW'
),`

const TMAP_BASE_FIELDS = `
SELECT 
	l.id AS link_id,
    l.url,
    l.submitted_by AS login_name,
    l.submit_date,
    COALESCE(puco.user_cats, l.global_cats) AS cats,
    COALESCE(puco.cats_from_user,0) AS cats_from_user,
    COALESCE(pus.user_summary, l.global_summary, '') AS summary,
    COALESCE(sc.summary_count, 0) AS summary_count,
    COALESCE(ts.times_starred, 0) AS times_starred,
	COALESCE(avs.avg_stars, 0) AS avg_stars,
	COALESCE(es.earliest_starrers, '') AS earliest_starrers,
	COALESCE(clc.click_count, 0) AS click_count,
    COALESCE(tc.tag_count, 0) AS tag_count,
    COALESCE(l.img_file, '') AS img_file`

const TMAP_FROM = LINKS_FROM

const TMAP_BASE_JOINS = `
LEFT JOIN PossibleUserCats_Other puco ON l.id = puco.link_id
LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id
LEFT JOIN TimesStarred ts ON l.id = ts.link_id
LEFT JOIN AverageStars avs ON l.id = avs.link_id
LEFT JOIN EarliestStarrers es ON l.id = es.link_id
LEFT JOIN ClickCount clc ON l.id = clc.link_id
LEFT JOIN TagCount tc ON l.id = tc.link_id
LEFT JOIN SummaryCount sc ON l.id = sc.link_id
`

const NSFW_JOINS = `
LEFT JOIN PossibleUserCats_NSFW puc ON l.id = puc.link_id
LEFT JOIN GlobalCatsFTS_NSFW gc ON l.id = gc.link_id`

const TMAP_NO_NSFW_CATS_WHERE = `WHERE l.id NOT IN (
	SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
	UNION
	SELECT link_id FROM user_cats_fts WHERE cats MATCH 'NSFW' AND submitted_by = ?
)`

var tmap_order_by_clauses = map[string]string{
	"times_starred": TMAP_ORDER_BY_TIMES_STARRED,
	"avg_stars":     TMAP_ORDER_BY_AVG_STARS,
	"newest":        TMAP_ORDER_BY_NEWEST,
	"oldest":        TMAP_ORDER_BY_OLDEST,
	"clicks":        TMAP_ORDER_BY_CLICKS,
}

const TMAP_ORDER_BY_TIMES_STARRED = `
ORDER BY 
	ts.times_starred DESC, 
	avs.avg_stars DESC,
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, l.id DESC,
	l.submit_date DESC,
	l.id DESC;`

const TMAP_ORDER_BY_AVG_STARS = `
ORDER BY 
	avs.avg_stars DESC, 
	ts.times_starred DESC,
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, 
	l.submit_date DESC,
	l.id DESC`

const TMAP_ORDER_BY_NEWEST = `
ORDER BY 
	l.submit_date DESC, 
	ts.times_starred DESC, 
	avs.avg_stars DESC,
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, 
	l.id DESC;`

const TMAP_ORDER_BY_OLDEST = `
ORDER BY 
	l.submit_date ASC, 
	ts.times_starred DESC, 
	avs.avg_stars DESC,
	clc.click_count DESC,
	tc.tag_count DESC,
	sc.summary_count DESC, 
	l.id DESC;`

const TMAP_ORDER_BY_CLICKS = `
ORDER BY
	clc.click_count DESC, 
	ts.times_starred DESC, 
	avs.avg_stars DESC,
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
LEFT JOIN StarsAssigned sa ON l.id = sa.link_id
`



// Shared building blocks
const USER_CATS_CTE = `UserCats AS (
    SELECT link_id, cats as user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
)`
	
const USER_STARS_CTE = `UserStars AS (
	SELECT s.link_id
	FROM Stars s
	INNER JOIN Users u ON u.id = s.user_id
	WHERE u.login_name = ?
)`

const POSSIBLE_USER_CATS_CTE = `
PossibleUserCats_Other AS (
	SELECT 
		link_id, 
		cats AS user_cats,
		(cats IS NOT NULL) AS cats_from_user
	FROM user_cats_fts
	WHERE submitted_by = ?
)`

var POSSIBLE_USER_CATS_CTE_NO_CATS_FROM_USER = strings.Replace(
	POSSIBLE_USER_CATS_CTE,
	`,
	(cats IS NOT NULL) AS cats_from_user`,
	"",
	1,
)

const POSSIBLE_USER_SUMMARY_CTE = `
PossibleUserSummary AS (
	SELECT
		link_id, 
		text as user_summary
	FROM Summaries
	INNER JOIN Users u ON u.id = submitted_by
	WHERE u.login_name = ?
)`

var CAT_FILTER_CTES = strings.Replace(
	POSSIBLE_USER_CATS_CTE,
	"WHERE submitted_by = ?", `WHERE submitted_by = ?
	AND cats MATCH ?`, 
	1) + `,
GlobalCatsFTS_Other AS (
    SELECT
        link_id,
        global_cats
    FROM global_cats_fts
    WHERE global_cats MATCH ?
)`

const CAT_FILTER_JOINS = `
LEFT JOIN PossibleUserCats_Other puco ON l.id = puco.link_id
LEFT JOIN GlobalCatsFTS_Other gco ON l.id = gco.link_id`

const STARRED_JOIN = `
INNER JOIN UserStars us ON l.id = us.link_id`
