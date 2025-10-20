package query

import (
	"strings"

	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	mutil "github.com/julianlk522/modeep/model/util"
)

// LINKS
type TmapLinksQueryBuilder interface {
	FromOptions(opts *model.TmapOptions) TmapLinksQueryBuilder
	Build() *Query

	fromCatFilters(cat_filters []string) TmapLinksQueryBuilder
	fromNeuteredCatFilters(neutered_cat_filters []string) TmapLinksQueryBuilder
	asSignedInUser(user_id string) TmapLinksQueryBuilder
	sortBy(metric string) TmapLinksQueryBuilder
	includeNSFW() TmapLinksQueryBuilder
	duringPeriod(period string) TmapLinksQueryBuilder
	withSummaryContaining(snippet string) TmapLinksQueryBuilder
	withURLContaining(snippet string) TmapLinksQueryBuilder
	withURLLacking(snippet string) TmapLinksQueryBuilder
}

type TmapSubmitted struct {
	*Query
}
type TmapStarred struct {
	*Query
}
type TmapTagged struct {
	*Query
}

func (ts *TmapSubmitted) Build() *Query {
	return ts.Query
}
func (ts *TmapStarred) Build() *Query {
	return ts.Query
}
func (ts *TmapTagged) Build() *Query {
	return ts.Query
}

// SUBMITTED
func NewTmapSubmitted(login_name string) *TmapSubmitted {
	return &TmapSubmitted{
		Query: &Query{
			Text: "WITH " + TMAP_BASE_CTES + "\n" +
				NSFW_CATS_CTES + "\n" +
				POSSIBLE_USER_CATS_CTE + "\n," +
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
			},
		},
	}
}

const SUBMITTED_AND = `
AND l.submitted_by = ?`

func (ts *TmapSubmitted) FromOptions(opts *model.TmapOptions) TmapLinksQueryBuilder {
	if len(opts.CatFiltersWithSpellingVariants) > 0 {
		ts.fromCatFilters(opts.CatFiltersWithSpellingVariants)
	}
	if len(opts.NeuteredCatFiltersWithSpellingVariants) > 0 {
		ts.fromNeuteredCatFilters(opts.NeuteredCatFiltersWithSpellingVariants)
	}
	if opts.AsSignedInUser != "" {
		ts.asSignedInUser(opts.AsSignedInUser)
	}
	if opts.SortBy != "" {
		ts.sortBy(opts.SortBy)
	}
	if opts.IncludeNSFW {
		ts.includeNSFW()
	}
	if opts.Period != "" {
		ts.duringPeriod(opts.Period)
	}
	if opts.SummaryContains != "" {
		ts.withSummaryContaining(opts.SummaryContains)
	}
	if opts.URLContains != "" {
		ts.withURLContaining(opts.URLContains)
	}
	if opts.URLLacks != "" {
		ts.withURLLacking(opts.URLLacks)
	}
	return ts
}

func (ts *TmapSubmitted) fromCatFilters(cat_filters []string) TmapLinksQueryBuilder {
	// Add CTEs
	ts.Text = strings.Replace(
		ts.Text,
		POSSIBLE_USER_CATS_CTE,
		"\n" + TMAP_CAT_FILTERS_CTES,
		1,
	)
	// Add JOINs
	ts.Text = strings.Replace(
		ts.Text,
		"LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id",
		TMAP_CAT_FILTERS_JOINS,
		1,
	)
	// Edit fields
	ts.Text = strings.Replace(
		ts.Text,
		TMAP_BASE_FIELDS,
		TMAP_FROM_CATS_FIELDS,
		1,
	)
	// Add AND condition
	ts.Text = strings.Replace(
		ts.Text,
		SUBMITTED_AND,
		"\n" + TMAP_CAT_FILTERS_AND + SUBMITTED_AND,
		1,
	)

	// Build match arg
	// cat_filters already adapted to include spelling variants in
	// GetTmapOptsFromRequestParams()
	match_arg := cat_filters[0]
	for i := 1; i < len(cat_filters); i++ {
		match_arg += " AND " + cat_filters[i]
	}

	// Add args: match_arg x2, login_name
	// old: [EARLIEST_STARRERS_LIMIT, login_name x 4]
	// new: [EARLIEST_STARRERS_LIMIT, login_name x 3,
	// match_arg x 2, login_name x 2]
	// so can insert before last 2
	login_name := ts.Args[1]
	up_to_last_2_args := ts.Args[:len(ts.Args) - 2]
	last_2_args := ts.Args[len(ts.Args) - 2:]

	new_args := make([]any, 0, len(ts.Args) + 3)
	new_args = append(new_args, up_to_last_2_args...)
	new_args = append(new_args, login_name, match_arg, match_arg)
	new_args = append(new_args, last_2_args...)

	ts.Args = new_args
	return ts
}

func (ts *TmapSubmitted) fromNeuteredCatFilters(neutered_cat_filters []string) TmapLinksQueryBuilder {
	// Build IN clause
	in_clause := "WHERE LOWER(cat) IN (?"
	for i := 1; i < len(neutered_cat_filters); i++ {
		in_clause += ", ?"
	}
	in_clause += ")"

	// Build and add CTEs
	neutered_cat_filters_ctes := strings.Replace(
		TMAP_NEUTERED_CAT_FILTERS_CTES,
		"WHERE LOWER(cat) IN (?)",
		in_clause,
		1,
	)
	// (after cat filters CTEs so that args come after)
	ts.Text = strings.Replace(
		ts.Text,
		POSSIBLE_USER_SUMMARY_CTE,
		POSSIBLE_USER_SUMMARY_CTE + ",\n" + neutered_cat_filters_ctes,
		1,
	)
	// Add AND condition
	ts.Text = strings.Replace(
		ts.Text,
		SUBMITTED_AND,
		"\n" + TMAP_NEUTERED_CAT_FILTERS_AND + SUBMITTED_AND,
		1,
	)

	// Add args: {neutered_cat_filters...}
	// Since we use IN, not FTS MATCH, casing matters and spelling variants
	// are not needed.
	neutered_cat_filters_args := make([]any, len(neutered_cat_filters))
	for i, cat := range neutered_cat_filters {
		neutered_cat_filters_args[i] = strings.ToLower(cat)
	}

	// old: [EARLIEST_STARRERS_LIMIT, login_name x 4]
	// new: [EARLIEST_STARRERS_LIMIT, login_name x 3,
	// neutered_cat_filters..., login_name]

	// OR if .fromCatFilters called first:

	// old: [EARLIEST_STARRERS_LIMIT, login_name x 3,
	// cat_filters x 2, login_name x 2]
	// new: [EARLIEST_STARRERS_LIMIT, login_name x 3,
	// cat_filters x 2, login_name, neutered_cat_filters..., login_name]

	// so can insert 2nd-to-last before login_name
	login_name := ts.Args[1]

	ts.Args = append(ts.Args[:len(ts.Args) - 1], neutered_cat_filters_args...)
	ts.Args = append(ts.Args, login_name)

	return ts
}

func (ts *TmapSubmitted) asSignedInUser(user_id string) TmapLinksQueryBuilder {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES + TMAP_AUTH_CTE + ",",
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS + TMAP_AUTH_FIELD,
		// in case TMAP_BASE_FIELDS are modified by .fromCatFilters
		TMAP_FROM_CATS_FIELDS, TMAP_FROM_CATS_FIELDS + TMAP_AUTH_FIELD,
		TMAP_BASE_JOINS, TMAP_BASE_JOINS + TMAP_AUTH_JOIN,
		// in case TMAP_BASE_JOINS are modified by .fromCatFilters
		TMAP_CAT_FILTERS_JOINS, TMAP_CAT_FILTERS_JOINS + TMAP_AUTH_JOIN,
	)
	ts.Text = fields_replacer.Replace(ts.Text)

	// Add args after first index
	new_args := make([]any, 0, len(ts.Args) + 1)

	first_arg := ts.Args[0]
	trailing_args := ts.Args[1:]

	new_args = append(new_args, first_arg, user_id)
	new_args = append(new_args, trailing_args...)

	ts.Args = new_args
	return ts
}

func (ts *TmapSubmitted) includeNSFW() TmapLinksQueryBuilder {
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
		TMAP_CAT_FILTERS_AND,
	) {
		cat_filter_where := strings.Replace(
			TMAP_CAT_FILTERS_AND,
			"AND",
			"WHERE",
			1,
		)
		ts.Text = strings.Replace(
			ts.Text,
			TMAP_CAT_FILTERS_AND,
			cat_filter_where,
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

	return ts
}

func (ts *TmapSubmitted) sortBy(metric string) TmapLinksQueryBuilder {
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

func (ts *TmapSubmitted) duringPeriod(period string) TmapLinksQueryBuilder {
	if period == "all" {
		return ts
	}

	period_clause, err := getPeriodClause(period)
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

func (ts *TmapSubmitted) withSummaryContaining(snippet string) TmapLinksQueryBuilder {
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

func (ts *TmapSubmitted) withURLContaining(snippet string) TmapLinksQueryBuilder {
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

func (ts *TmapSubmitted) withURLLacking(snippet string) TmapLinksQueryBuilder {
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

// STARRED
func NewTmapStarred(login_name string) *TmapStarred {
	q := &TmapStarred{
		Query: &Query{
			Text: "WITH " + TMAP_BASE_CTES + "\n" +
				NSFW_CATS_CTES + "\n" +
				USER_STARS_CTE + ",\n" +
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
			},
		},
	}

	return q
}

const STARRED_AND = ` 
AND l.submitted_by != ?`

func (ts *TmapStarred) FromOptions(opts *model.TmapOptions) TmapLinksQueryBuilder {
	if len(opts.CatFiltersWithSpellingVariants) > 0 {
		ts.fromCatFilters(opts.CatFiltersWithSpellingVariants)
	}
	if len(opts.NeuteredCatFiltersWithSpellingVariants) > 0 {
		ts.fromNeuteredCatFilters(opts.NeuteredCatFiltersWithSpellingVariants)
	}
	if opts.AsSignedInUser != "" {
		ts.asSignedInUser(opts.AsSignedInUser)
	}
	if opts.SortBy != "" {
		ts.sortBy(opts.SortBy)
	}
	if opts.IncludeNSFW {
		ts.includeNSFW()
	}
	if opts.Period != "" {
		ts.duringPeriod(opts.Period)
	}
	if opts.SummaryContains != "" {
		ts.withSummaryContaining(opts.SummaryContains)
	}
	if opts.URLContains != "" {
		ts.withURLContaining(opts.URLContains)
	}
	if opts.URLLacks != "" {
		ts.withURLLacking(opts.URLLacks)
	}
	return ts
}

func (ts *TmapStarred) fromCatFilters(cat_filters []string) TmapLinksQueryBuilder {
	// Add CTEs
	ts.Text = strings.Replace(
		ts.Text,
		POSSIBLE_USER_CATS_CTE,
		"\n" + TMAP_CAT_FILTERS_CTES,
		1,
	)
	// Add JOINs
	ts.Text = strings.Replace(
		ts.Text,
		"LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id",
		TMAP_CAT_FILTERS_JOINS,
		1,
	)
	// Edit fields
	ts.Text = strings.Replace(
		ts.Text,
		TMAP_BASE_FIELDS,
		TMAP_FROM_CATS_FIELDS,
		1,
	)
	// Add AND condition
	ts.Text = strings.Replace(
		ts.Text,
		TMAP_NO_NSFW_CATS_WHERE,
		TMAP_NO_NSFW_CATS_WHERE + "\n" + TMAP_CAT_FILTERS_AND,
		1,
	)

	// Build match arg
	match_arg := cat_filters[0]
	for i := 1; i < len(cat_filters); i++ {
		match_arg += " AND " + cat_filters[i]
	}

	// Add args: match_arg x2, login_name
	// old: [EARLIEST_STARRERS_LIMIT, login_name x 5]
	// new: [EARLIEST_STARRERS_LIMIT, login_name x 4,
	// match_arg x 2, login_name x 2]
	// so insert before last 2
	login_name := ts.Args[1]
	last_2_args := ts.Args[len(ts.Args) - 2:]

	new_args := append([]any{}, ts.Args[:len(ts.Args) - 2]...)
	new_args = append(new_args, login_name, match_arg, match_arg)
	new_args = append(new_args, last_2_args...)
	ts.Args = new_args

	return ts
}

func (ts *TmapStarred) fromNeuteredCatFilters(neutered_cat_filters []string) TmapLinksQueryBuilder {
	if len(neutered_cat_filters) == 0 {
		return ts
	}

	// Build IN clause
	in_clause := "WHERE LOWER(cat) IN (?"
	for i := 1; i < len(neutered_cat_filters); i++ {
		in_clause += ", ?"
	}
	in_clause += ")"

	// Build and add CTEs
	neutered_cat_filters_ctes := strings.Replace(
		TMAP_NEUTERED_CAT_FILTERS_CTES,
		"WHERE LOWER(cat) IN (?)",
		in_clause,
		1,
	)
	// (after cat filters CTEs so that args come after)
	ts.Text = strings.Replace(
		ts.Text,
		POSSIBLE_USER_SUMMARY_CTE,
		POSSIBLE_USER_SUMMARY_CTE + ",\n" + neutered_cat_filters_ctes,
		1,
	)
	// Add AND condition
	ts.Text = strings.Replace(
		ts.Text,
		STARRED_AND,
		"\n" + TMAP_NEUTERED_CAT_FILTERS_AND + STARRED_AND,
		1,
	)

	// Add args: {neutered_cat_filters...}
	// Since we use IN, not FTS MATCH, casing matters and spelling variants
	// are not needed.
	neutered_cat_filters_args := make([]any, len(neutered_cat_filters))
	for i, cat := range neutered_cat_filters {
		neutered_cat_filters_args[i] = strings.ToLower(cat)
	}

	// old: [EARLIEST_STARRERS_LIMIT, login_name x 5]
	// new: [EARLIEST_STARRERS_LIMIT, login_name x 4, 
	// neutered_cat_filters..., login_name]

	// OR if .fromCatFilters called first:

	// old: [EARLIEST_STARRERS_LIMIT, login_name x 4,
	// cat_filters x 2, login_name x 2]
	// new: [EARLIEST_STARRERS_LIMIT, login_name x 3,
	// cat_filters x 2, login_name, neutered_cat_filters..., login_name]

	// so can insert 2nd-to-last before login_name
	login_name := ts.Args[1]

	ts.Args = append(ts.Args[:len(ts.Args) - 1], neutered_cat_filters_args...)
	ts.Args = append(ts.Args, login_name)

	return ts
}

func (ts *TmapStarred) asSignedInUser(req_user_id string) TmapLinksQueryBuilder {
	fields_replacer := strings.NewReplacer(
		TMAP_BASE_CTES, TMAP_BASE_CTES + TMAP_AUTH_CTE + ",",
		TMAP_BASE_FIELDS, TMAP_BASE_FIELDS + TMAP_AUTH_FIELD,
		// in case TMAP_BASE_FIELDS are modified by .fromCatFilters
		TMAP_FROM_CATS_FIELDS, TMAP_FROM_CATS_FIELDS + TMAP_AUTH_FIELD,
		TMAP_BASE_JOINS, TMAP_BASE_JOINS + TMAP_AUTH_JOIN,
		// in case TMAP_BASE_JOINS are modified by .fromCatFilters
		TMAP_CAT_FILTERS_JOINS, TMAP_CAT_FILTERS_JOINS + TMAP_AUTH_JOIN,
	)

	ts.Text = fields_replacer.Replace(ts.Text)

	// Add args: req_user_id
	// old: [EARLIEST_STARRERS_LIMIT login_name x 3,
	// match_arg x 2, login_name x 2] (if cat filter applied)
	// new: [EARLIEST_STARRERS_LIMIT, req_user_id, login_name x 3,
	// match_arg x 2, login_name x 2]
	// so insert after EARLIEST_STARRERS_LIMIT
	first_arg := ts.Args[0]
	trailing_args := ts.Args[1:]

	new_args := make([]any, 0, len(ts.Args) + 1)
	new_args = append(new_args, first_arg, req_user_id)
	new_args = append(new_args, trailing_args...)

	ts.Args = new_args
	return ts
}

func (ts *TmapStarred) includeNSFW() TmapLinksQueryBuilder {
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
		TMAP_CAT_FILTERS_AND,
	) {
		cat_filter_where := strings.Replace(
			TMAP_CAT_FILTERS_AND,
			"AND",
			"WHERE",
			1,
		)
		ts.Text = strings.Replace(
			ts.Text,
			TMAP_CAT_FILTERS_AND,
			cat_filter_where,
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

	return ts
}

func (ts *TmapStarred) sortBy(metric string) TmapLinksQueryBuilder {
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

func (ts *TmapStarred) duringPeriod(period string) TmapLinksQueryBuilder {
	if period == "all" {
		return ts
	}

	period_clause, err := getPeriodClause(period)
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

func (ts *TmapStarred) withSummaryContaining(snippet string) TmapLinksQueryBuilder {
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

func (ts *TmapStarred) withURLContaining(snippet string) TmapLinksQueryBuilder {
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

func (ts *TmapStarred) withURLLacking(snippet string) TmapLinksQueryBuilder {
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

// TAGGED
func NewTmapTagged(login_name string) *TmapTagged {
	q := &TmapTagged{
		Query: &Query{
			Text: "WITH " +
				TMAP_BASE_CTES + "\n" +
				NSFW_CATS_CTES + "\n" +
				USER_CATS_CTE + ",\n" +
				USER_STARS_CTE + ",\n" +
				POSSIBLE_USER_SUMMARY_CTE +
				TAGGED_FIELDS +
				TMAP_FROM +
				TAGGED_JOINS +
				NSFW_JOINS + "\n" +
				TAGGED_NO_NSFW_CATS_WHERE +
				TAGGED_AND +
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

	q.Text = strings.ReplaceAll(q.Text, "LOGIN_NAME", login_name)
	return q
}

const USER_CATS_CTE = `UserCats AS (
    SELECT link_id, cats as user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
)`

var TAGGED_FIELDS = strings.Replace(
	strings.Replace(
		TMAP_BASE_FIELDS,
		"COALESCE(puca.user_cats, l.global_cats) AS cats",
		"uct.user_cats",
		1,
	),
	`COALESCE(puca.cats_from_user, 0) AS cats_from_user`,
	"1 AS cats_from_user",
	1,
)

var TAGGED_JOINS = strings.Replace(
	TMAP_BASE_JOINS,
	"LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id",
	"INNER JOIN UserCats uct ON l.id = uct.link_id",
	1,
) + strings.Replace(
	STARRED_JOIN,
	"INNER",
	"LEFT",
	1,
)

var TAGGED_NO_NSFW_CATS_WHERE = strings.ReplaceAll(
	TMAP_NO_NSFW_CATS_WHERE,
	"puca.user_cats",
	"uct.user_cats",
)

func (tt *TmapTagged) FromOptions(opts *model.TmapOptions) TmapLinksQueryBuilder {
	if len(opts.CatFiltersWithSpellingVariants) > 0 {
		tt.fromCatFilters(opts.CatFiltersWithSpellingVariants)
	}
	if len(opts.NeuteredCatFiltersWithSpellingVariants) > 0 {
		tt.fromNeuteredCatFilters(opts.NeuteredCatFiltersWithSpellingVariants)
	}
	if opts.AsSignedInUser != "" {
		tt.asSignedInUser(opts.AsSignedInUser)
	}
	if opts.SortBy != "" {
		tt.sortBy(opts.SortBy)
	}
	if opts.IncludeNSFW {
		tt.includeNSFW()
	}
	if opts.Period != "" {
		tt.duringPeriod(opts.Period)
	}
	if opts.SummaryContains != "" {
		tt.withSummaryContaining(opts.SummaryContains)
	}
	if opts.URLContains != "" {
		tt.withURLContaining(opts.URLContains)
	}
	if opts.URLLacks != "" {
		tt.withURLLacking(opts.URLLacks)
	}
	return tt
}

func (tt *TmapTagged) fromCatFilters(cat_filters []string) TmapLinksQueryBuilder {
	if len(cat_filters) == 0 || cat_filters[0] == "" {
		return tt
	}

	// Add MATCH clause
	match_clause := `
	AND uct.user_cats MATCH ?`

	// .sortBy hasn't been called yet - can safely assume
	// it will have this one
	tt.Text = strings.Replace(
		tt.Text,
		TMAP_ORDER_BY_TIMES_STARRED,
		match_clause + TMAP_ORDER_BY_TIMES_STARRED,
		1,
	)

	// Append arg
	match_arg := cat_filters[0]
	for i := 1; i < len(cat_filters); i++ {
		match_arg += " AND " + cat_filters[i]
	}
	tt.Args = append(tt.Args, match_arg)

	return tt
}

func (tt *TmapTagged) fromNeuteredCatFilters(neutered_cat_filters []string) TmapLinksQueryBuilder {
	if len(neutered_cat_filters) == 0 {
		return tt
	}

	// Build IN clause
	in_clause := "WHERE LOWER(cat) IN (?"
	for i := 1; i < len(neutered_cat_filters); i++ {
		in_clause += ", ?"
	}
	in_clause += ")"

	// Build and add CTEs
	neutered_cat_filters_ctes := strings.Replace(
		TAGGED_NEUTERED_CAT_FILTER_CTES,
		"WHERE LOWER(cat) IN (?)",
		in_clause,
		1,
	)

	// Add CTEs
	tt.Text = strings.Replace(
		tt.Text,
		TAGGED_FIELDS,
		",\n" + neutered_cat_filters_ctes + TAGGED_FIELDS,
		1,
	)
	
	// Add AND condition
	// .sortBy hasn't been called yet - can safely assume
	// it will have this one
	tt.Text = strings.Replace(
		tt.Text,
		TMAP_ORDER_BY_TIMES_STARRED,
		"\n" + TMAP_NEUTERED_CAT_FILTERS_AND + TMAP_ORDER_BY_TIMES_STARRED,
		1,
	)
	// this works a bit better than replacing TAGGED_AND since it arranges the
	// query such that args can be added at the end of the slice.

	// Add args: {neutered_cat_filters...}
	// Since we use IN, not FTS MATCH, casing matters and spelling variants
	// are not needed.
	neutered_cat_filters_args := make([]any, len(neutered_cat_filters))
	for i, cat := range neutered_cat_filters {
		neutered_cat_filters_args[i] = strings.ToLower(cat)
	}

	// old: [EARLIEST_STARRERS_LIMIT, login_name x 5]
	// new: [EARLIEST_STARRERS_LIMIT, login_name x 4, 
	// neutered_cat_filters..., login_name]

	// OR if .fromCatFilters called first:

	// old: [EARLIEST_STARRERS_LIMIT, login_name x 4, match_arg, login_name]
	// new: [EARLIEST_STARRERS_LIMIT, login_name x 4, match_arg, 
	// neutered_cat_filters..., login_name]

	// so can insert in 2nd-to-last position before login_name
	login_name := tt.Args[len(tt.Args) - 1]
	tt.Args = append(tt.Args[:len(tt.Args) - 1], neutered_cat_filters_args...)
	tt.Args = append(tt.Args, login_name)
	
	return tt
}

const TAGGED_NEUTERED_CAT_FILTER_CTES = `UserCatsSplit(link_id, cat, str) AS (
    SELECT link_id, '', user_cats||','
    FROM UserCats
    UNION ALL SELECT
        link_id,
        substr(str, 0, instr(str, ',')),
        substr(str, instr(str, ',') + 1)
    FROM UserCatsSplit
    WHERE str != ''
),
ExcludedLinksDueToNeutering AS (
	SELECT link_id
	FROM UserCatsSplit
	WHERE LOWER(cat) IN (?)
)`

func (tt *TmapTagged) asSignedInUser(req_user_id string) TmapLinksQueryBuilder {
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

func (tt *TmapTagged) includeNSFW() TmapLinksQueryBuilder {
	// Remove NSFW clause
	tt.Text = strings.Replace(
		tt.Text,
		TAGGED_NO_NSFW_CATS_WHERE,
		"",
		1,
	)

	// Swap following condition keyword
	if strings.Contains(
		tt.Text,
		TMAP_CAT_FILTERS_AND,
	) {
		cat_filter_where := strings.Replace(
			TMAP_CAT_FILTERS_AND,
			"AND",
			"WHERE",
			1,
		)
		tt.Text = strings.Replace(
			tt.Text,
			TMAP_CAT_FILTERS_AND,
			cat_filter_where,
			1,
		)
	} else {
		tagged_where := strings.Replace(
			TAGGED_AND,
			"AND",
			"WHERE",
			1,
		)
		tt.Text = strings.Replace(
			tt.Text,
			TAGGED_AND,
			tagged_where,
			1,
		)
	}

	return tt
}

func (tt *TmapTagged) sortBy(metric string) TmapLinksQueryBuilder {
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

func (tt *TmapTagged) duringPeriod(period string) TmapLinksQueryBuilder {
	if period == "all" {
		return tt
	}

	period_clause, err := getPeriodClause(period)
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

func (tt *TmapTagged) withSummaryContaining(snippet string) TmapLinksQueryBuilder {
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

func (tt *TmapTagged) withURLContaining(snippet string) TmapLinksQueryBuilder {
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

func (tt *TmapTagged) withURLLacking(snippet string) TmapLinksQueryBuilder {
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

// LINKS SHARED BUILDING BLOCKS
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

const TMAP_BASE_FIELDS = `
SELECT 
	l.id AS link_id,
    l.url,
    l.submitted_by AS login_name,
    l.submit_date,
    COALESCE(puca.user_cats, l.global_cats) AS cats,
    COALESCE(puca.cats_from_user, 0) AS cats_from_user,
    COALESCE(pus.user_summary, l.global_summary, '') AS summary,
    COALESCE(sc.summary_count, 0) AS summary_count,
    COALESCE(ts.times_starred, 0) AS times_starred,
	COALESCE(avs.avg_stars, 0) AS avg_stars,
	COALESCE(es.earliest_starrers, '') AS earliest_starrers,
	COALESCE(clc.click_count, 0) AS click_count,
    COALESCE(tc.tag_count, 0) AS tag_count,
    COALESCE(l.img_file, '') AS img_file`

const TMAP_FROM_CATS_FIELDS = `
SELECT 
	l.id AS link_id,
    l.url,
    l.submitted_by AS login_name,
    l.submit_date,
    COALESCE(pucmrp.user_cats, l.global_cats) AS cats,
    COALESCE(pucmrp.cats_from_user, 0) AS cats_from_user,
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
LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id
LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id
LEFT JOIN TimesStarred ts ON l.id = ts.link_id
LEFT JOIN AverageStars avs ON l.id = avs.link_id
LEFT JOIN EarliestStarrers es ON l.id = es.link_id
LEFT JOIN ClickCount clc ON l.id = clc.link_id
LEFT JOIN TagCount tc ON l.id = tc.link_id
LEFT JOIN SummaryCount sc ON l.id = sc.link_id`


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
	COALESCE(sa.stars_assigned, 0) AS stars_assigned`

const TMAP_AUTH_JOIN = `
LEFT JOIN StarsAssigned sa ON l.id = sa.link_id
`

// Other shared (links queries only)
const POSSIBLE_USER_CATS_CTE = `PossibleUserCatsAny AS (
	SELECT 
		link_id, 
		cats AS user_cats,
		(cats IS NOT NULL) AS cats_from_user
	FROM user_cats_fts
	WHERE submitted_by = ?
)`

const POSSIBLE_USER_SUMMARY_CTE = `PossibleUserSummary AS (
	SELECT
		link_id, 
		text as user_summary
	FROM Summaries
	INNER JOIN Users u ON u.id = submitted_by
	WHERE u.login_name = ?
)`

const STARRED_JOIN = `
INNER JOIN UserStars us ON l.id = us.link_id`

// NSFW LINKS COUNT
// Communicates how many NSFW links are hidden if the user does not
// opt into seeing them via the "?nsfw=true" URL params.
// This clarifies the view somewhat when cat counts given via spellfix matches
// or Top Cats totals appear to not equal the amount of matching links
// (because some are hidden).
type TmapNSFWLinksCount struct {
	*Query
}

func NewTmapNSFWLinksCount(login_name string) *TmapNSFWLinksCount {
	return &TmapNSFWLinksCount{
		&Query{
			Text: "WITH " + POSSIBLE_USER_CATS_CTE_NO_CATS_FROM_USER + ",\n" +
				NSFW_CATS_CTES + "\n" +
				USER_STARS_CTE + `
			SELECT count(*) as NSFW_link_count
				FROM Links l` + "\n" +
				"LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id" +
				NSFW_JOINS + "\n" +
				NSFW_LINKS_COUNT_WHERE +
				NSFW_LINKS_COUNT_FINAL_AND + ";",
			Args: []any{
				login_name,
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
			tnlc.submittedOnly()
		case "starred":
			tnlc.starredOnly()
		case "tagged":
			tnlc.taggedOnly()
		default:
			tnlc.Error = e.ErrInvalidOnlySectionParams
			return tnlc
		}
	}
	if len(opts.CatFiltersWithSpellingVariants) > 0 {
		tnlc.fromCatFilters(opts.CatFiltersWithSpellingVariants)
	}
	if opts.Period != "" {
		tnlc.duringPeriod(opts.Period)
	}
	if opts.SummaryContains != "" {
		tnlc.withSummaryContaining(opts.SummaryContains)
	}
	if opts.URLContains != "" {
		tnlc.withURLContaining(opts.URLContains)
	}
	if opts.URLLacks != "" {
		tnlc.withURLLacking(opts.URLLacks)
	}
	return tnlc
}

func (tnlc *TmapNSFWLinksCount) submittedOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		NSFW_LINKS_COUNT_FINAL_AND,
		"\nAND l.submitted_by = ?",
		1,
	)

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) starredOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		NSFW_LINKS_COUNT_FINAL_AND,
		"\nAND l.id IN (SELECT link_id FROM UserStars)",
		1,
	)
	// since NSFW_LINKS_COUNT_FINAL_AND had an arg placeholder,
	// we remove the corresponding arg
	tnlc.Args = tnlc.Args[:len(tnlc.Args)-1]

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) taggedOnly() *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		NSFW_LINKS_COUNT_FINAL_AND,
		TAGGED_AND,
		1,
	)

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) fromCatFilters(cat_filters []string) *TmapNSFWLinksCount {
	if len(cat_filters) == 0 || cat_filters[0] == "" {
		return tnlc
	}

	// Add CTEs
	cat_filter_ctes_no_cats_from_user := strings.Replace(
		TMAP_CAT_FILTERS_CTES,
		`,
		(cats IS NOT NULL) AS cats_from_user`,
		"",
		1,
	)
	tnlc.Text = strings.Replace(
		tnlc.Text,
		POSSIBLE_USER_CATS_CTE_NO_CATS_FROM_USER,
		cat_filter_ctes_no_cats_from_user,
		1,
	)

	// Add JOINs
	tnlc.Text = strings.Replace(
		tnlc.Text,
		"LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id",
		TMAP_CAT_FILTERS_JOINS,
		1,
	)

	// Add user/global cat filter condition
	tnlc.Text = strings.Replace(
		tnlc.Text,
		NSFW_LINKS_COUNT_WHERE,
		NSFW_LINKS_COUNT_WHERE + "\n" + TMAP_CAT_FILTERS_AND,
		1,
	)

	// Build MATCH clause
	// (cat_filters already include spelling variants via
	// GetTmapOptsFromRequestParams())
	match_arg := cat_filters[0]
	for i := 1; i < len(cat_filters); i++ {
		match_arg += " AND " + cat_filters[i]
	}

	// Add args: login_name, match arg x2
	new_args := make([]any, 0, len(tnlc.Args) + 3)
	// since other methods push new args to end of slice,
	// better to insert these from the start (always after the first 2,
	// which are login_name x2)
	login_name := tnlc.Args[0]
	first_2_args := tnlc.Args[:2]
	trailing_args := tnlc.Args[2:]

	new_args = append(new_args, first_2_args...)
	new_args = append(new_args, login_name, match_arg, match_arg)
	new_args = append(new_args, trailing_args...)

	tnlc.Args = new_args
	return tnlc
}

func (tnlc *TmapNSFWLinksCount) duringPeriod(period string) *TmapNSFWLinksCount {
	if period == "all" {
		return tnlc
	}

	period_clause, err := getPeriodClause(period)
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

// As with cat filters, the specified snippet must either be present on the user's
// summary for a link, if they submitted one, or otherwise on the global summary
func (tnlc *TmapNSFWLinksCount) withSummaryContaining(snippet string) *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		POSSIBLE_USER_CATS_CTE_NO_CATS_FROM_USER,
		POSSIBLE_USER_CATS_CTE_NO_CATS_FROM_USER + ",\n" + POSSIBLE_USER_SUMMARY_CTE,
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
		"\nAND COALESCE(pus.user_summary,l.global_summary) LIKE ?;",
		1,
	)

	// Prepend login_name arg for POSSIBLE_USER_SUMMARY_CTE
	// (always first arg for TmapNSFWLinksCount queries)
	login_name := tnlc.Args[0]
	tnlc.Args = append([]any{login_name}, tnlc.Args...)
	// Append summary snippet arg
	tnlc.Args = append(tnlc.Args, "%" + snippet + "%")

	return tnlc
}

func (tnlc *TmapNSFWLinksCount) withURLContaining(snippet string) *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		";",
		"\nAND url LIKE ?;",
		1,
	)
	tnlc.Args = append(tnlc.Args, "%" + snippet + "%")
	return tnlc
}

func (tnlc *TmapNSFWLinksCount) withURLLacking(snippet string) *TmapNSFWLinksCount {
	tnlc.Text = strings.Replace(
		tnlc.Text,
		";",
		"\nAND url NOT LIKE ?;",
		1,
	)
	tnlc.Args = append(tnlc.Args, "%" + snippet + "%")
	return tnlc
}

// SHARED BUILDING BLOCKS FOR LINKS AND NSFW COUNT QUERIES
const NSFW_CATS_CTES = `PossibleUserCatsNSFW AS (
    SELECT 
		link_id, 
		cats AS user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
	AND cats MATCH 'NSFW'
),
GlobalNSFWCats AS (
	SELECT
		link_id,
		global_cats
	FROM global_cats_fts
	WHERE global_cats MATCH 'NSFW'
),`

const NSFW_JOINS = `
LEFT JOIN PossibleUserCatsNSFW pucnsfw ON l.id = pucnsfw.link_id
LEFT JOIN GlobalNSFWCats gnsfwc ON l.id = gnsfwc.link_id`

// As with cat filters and summary filters on user Treasure Map pages,
// NSFW filters are made against the Treasure Map owner's input if it exists,
// otherwise global tags/cats. Links are considered NSFW (included in the count
// and hidden by default) if either the Treasure Map owner tagged it "NSFW"
// OR they have not tagged at all but the global tag contains "NSFW."
// If the Treasure Map owner has tagged a link and _not_ assigned NSFW
// as a cat in their tag, the link is visible by default.
const NSFW_LINKS_COUNT_WHERE = `WHERE (
	(puca.user_cats IS NULL AND gnsfwc.global_cats IS NOT NULL)
	OR
	pucnsfw.user_cats IS NOT NULL
)`

const NSFW_LINKS_COUNT_FINAL_AND = `
AND (
	l.submitted_by = ?
	OR l.id IN (SELECT link_id FROM UserStars)
	OR l.id IN 
		(
		SELECT link_id
		FROM PossibleUserCatsNSFW
		)
	)`

// Links which the Treasure Map owner has tagged "NSFW" are hidden by default.
// Additionally, if they have not tagged one but its global tag contains "NSFW,"
// it is hidden by default.
const TMAP_NO_NSFW_CATS_WHERE = `WHERE (
	(puca.user_cats IS NULL AND gnsfwc.global_cats IS NULL)
	OR
	(puca.user_cats IS NOT NULL AND pucnsfw.user_cats IS NULL)
)`

var POSSIBLE_USER_CATS_CTE_NO_CATS_FROM_USER = strings.Replace(
	POSSIBLE_USER_CATS_CTE,
	`,
		(cats IS NOT NULL) AS cats_from_user
`,
	"\n",
	1,
)

const USER_STARS_CTE = `UserStars AS (
	SELECT s.link_id
	FROM Stars s
	INNER JOIN Users u ON u.id = s.user_id
	WHERE u.login_name = ?
)`

const TAGGED_AND = `
AND l.submitted_by != ?
AND l.id NOT IN
	(SELECT link_id FROM UserStars)`

var TMAP_CAT_FILTERS_CTES = POSSIBLE_USER_CATS_CTE + `,
PossibleUserCatsMatchingRequestParams AS (
	SELECT 
		link_id,
		cats AS user_cats,
		1 AS cats_from_user
	FROM user_cats_fts
	WHERE submitted_by = ?
	AND cats MATCH ?
),
GlobalCatsMatchingRequestParams AS (
    SELECT
        link_id,
        global_cats
    FROM global_cats_fts
    WHERE global_cats MATCH ?
)`

const TMAP_CAT_FILTERS_JOINS = `
LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id
LEFT JOIN PossibleUserCatsMatchingRequestParams pucmrp ON l.id = pucmrp.link_id
LEFT JOIN GlobalCatsMatchingRequestParams gcmrp ON l.id = gcmrp.link_id`

// Cats passed in filters are checked against user's assigned cats
// if they submitted a tag, otherwise the global tag.
const TMAP_CAT_FILTERS_AND = `AND (
	(puca.user_cats IS NULL AND gcmrp.global_cats IS NOT NULL)
	OR 
	pucmrp.user_cats IS NOT NULL
)`

const TMAP_NEUTERED_CAT_FILTERS_CTES = `FinalCatsValue AS (
	SELECT 
		l.id AS link_id,
		COALESCE(puca.user_cats, l.global_cats) AS cats
	FROM Links l
	LEFT JOIN PossibleUserCatsAny puca ON l.id = puca.link_id
),
FinalCatsSplit(link_id, cat, str) AS (
    SELECT link_id, '', cats||','
    FROM FinalCatsValue
    UNION ALL SELECT
        link_id,
        substr(str, 0, instr(str, ',')),
        substr(str, instr(str, ',') + 1)
    FROM FinalCatsSplit
    WHERE str != ''
),
ExcludedLinksDueToNeutering AS (
	SELECT link_id
	FROM FinalCatsSplit
	WHERE LOWER(cat) IN (?)
)`

const TMAP_NEUTERED_CAT_FILTERS_AND = "AND l.id NOT IN ExcludedLinksDueToNeutering"

// USER PROFILE
// (visible on Treasure Map when no filters applied and no individual section
// selected)
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
