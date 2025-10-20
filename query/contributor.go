package query

import (
	"net/url"
	"strings"
)

type Contributors struct {
	*Query

	// for consistent strings replaces
	hasWhereAfterFrom bool
}

func NewTopContributors() *Contributors {
	return (&Contributors{
		Query: &Query{
			Text: CONTRIBUTORS_BASE,
			Args: []any{CONTRIBUTORS_PAGE_LIMIT},
		},
	})
}

const CONTRIBUTORS_BASE = `SELECT
count(l.id) as count, l.submitted_by
FROM Links l
GROUP BY l.submitted_by
ORDER BY count DESC, l.submitted_by ASC
LIMIT ?;`

func (c *Contributors) FromRequestParams(params url.Values) *Contributors {
	cats_filter_params := params.Get("cats")
	if cats_filter_params != "" {
		cat_filters := GetCatsOptionalPluralOrSingularForms(strings.Split(cats_filter_params, ","))
		c = c.fromCatFilters(cat_filters)
	}
	neutered_params := params.Get("neutered")
	if neutered_params != "" {
		// Since we use IN, not FTS MATCH, spelling variants are not
		// needed and casing matters
		neutered_cat_filters := strings.Split(neutered_params, ",")
		c = c.fromNeuteredCatFilters(neutered_cat_filters)
	}
	summary_contains_params := params.Get("summary_contains")
	if summary_contains_params != "" {
		c = c.withGlobalSummaryContaining(summary_contains_params)
	}
	url_contains_params := params.Get("url_contains")
	if url_contains_params != "" {
		c = c.withURLContaining(url_contains_params)
	}
	url_lacks_params := params.Get("url_lacks")
	if url_lacks_params != "" {
		c = c.withURLLacking(url_lacks_params)
	}
	period_params := params.Get("period")
	if period_params != "" {
		c = c.duringPeriod(period_params)
	}
	return c
}

func (c *Contributors) fromCatFilters(cat_filters []string) *Contributors {
	if len(cat_filters) == 0 {
		return c
	}

	// Add CTE
	c.Text = "WITH " + CONTRIBUTORS_CAT_FILTERS_CTES + "\n" + c.Text

	// Add JOIN
	c.Text = strings.Replace(
		c.Text,
		"FROM Links l",
		"FROM Links l" + "\n" + CONTRIBUTORS_CAT_FILTERS_JOIN,
		1,
	)

	// Build MATCH arg
	// (spelling variations added in .FromRequestParams())
	match_arg := cat_filters[0]
	for i := 1; i < len(cat_filters); i++ {
		match_arg += " AND " + cat_filters[i]
	}
	
	// Add before LIMIT arg
	// old: [CONTRIBUTORS_PAGE_LIMIT]
	// new: [match_arg, CONTRIBUTORS_PAGE_LIMIT]
	c.Args = append(c.Args[:len(c.Args) - 1], match_arg, CONTRIBUTORS_PAGE_LIMIT)
	return c
}

const CONTRIBUTORS_CAT_FILTERS_CTES = LINKS_CATS_FILTER_CTE
const CONTRIBUTORS_CAT_FILTERS_JOIN = LINKS_CATS_FILTER_JOIN

func (c *Contributors) fromNeuteredCatFilters(neutered_cat_filters []string) *Contributors {
	if len(neutered_cat_filters) == 0 {
		return c
	}

	// Build IN clause
	in_clause := "WHERE LOWER(global_cat) IN (?"
	for i := 1; i < len(neutered_cat_filters); i++ {
		in_clause += ", ?"
	}
	in_clause += ")"

	// Build CTEs
	neutered_cat_filters_ctes := strings.Replace(
		CONTRIBUTORS_NEUTERED_CAT_FILTERS_CTES,
		"WHERE LOWER(global_cat) IN (?)",
		in_clause,
		1,
	)

	// Add CTEs
	// (first determine whether to add the "WITH" or if it was already added
	// by .fromCatFilters())
	if strings.Contains(c.Text, "WITH") {
		c.Text = strings.Replace(
			c.Text,
			CONTRIBUTORS_CAT_FILTERS_CTES,
			CONTRIBUTORS_CAT_FILTERS_CTES + ",\n" + neutered_cat_filters_ctes,
			1,
		)
	} else {
		c.Text = strings.Replace(
			c.Text,
			CONTRIBUTORS_BASE,
			"WITH " + neutered_cat_filters_ctes + "\n" + CONTRIBUTORS_BASE,
			1,
		)
	}

	// Insert WHERE condition
	c.Text = strings.Replace(
		c.Text,
		"GROUP BY l.submitted_by",
		CONTRIBUTORS_NEUTERED_CATS_WHERE + "\n" + "GROUP BY l.submitted_by",
		1,
	)

	// Add args: {neutered_cat_filters...}
	neutered_cat_filters_args := make([]any, len(neutered_cat_filters))
	for i, cat := range neutered_cat_filters {
		neutered_cat_filters_args[i] = strings.ToLower(cat) // casing matters
	}

	// old: [CONTRIBUTORS_PAGE_LIMIT]
	// new: [neutered_cat_filters..., CONTRIBUTORS_PAGE_LIMIT]

	// OR if .fromCatFilters() called first:
	
	// old: [cat_filters, CONTRIBUTORS_PAGE_LIMIT]
	// new: [cat_filters, neutered_cat_filters..., CONTRIBUTORS_PAGE_LIMIT]

	// Insert in front of LIMIT
	c.Args = append(c.Args[:len(c.Args) - 1], neutered_cat_filters_args...)
	c.Args = append(c.Args, CONTRIBUTORS_PAGE_LIMIT)

	return c
}

const CONTRIBUTORS_NEUTERED_CAT_FILTERS_CTES = LINKS_NEUTERED_CATS_FILTER_CTES
var CONTRIBUTORS_NEUTERED_CATS_WHERE = strings.Replace(
	LINKS_NEUTERED_CATS_AND,
	"AND",
	"WHERE",
	1,
)

func (c *Contributors) withGlobalSummaryContaining(snippet string) *Contributors {
	clause_keyword := "WHERE"
	if c.hasWhereAfterFrom {
		clause_keyword = "AND"
	} else {
		c.hasWhereAfterFrom = true
	}
	c.Text = strings.Replace(
		c.Text,
		"GROUP BY l.submitted_by",
		clause_keyword + " global_summary LIKE ?\nGROUP BY l.submitted_by",
		1,
	)

	// Add arg in 2nd-to-last position before LIMIT
	last_arg := c.Args[len(c.Args) - 1]
	c.Args = append(c.Args[:len(c.Args) - 1], "%" + snippet + "%", last_arg)

	return c
}

func (c *Contributors) withURLContaining(snippet string) *Contributors {
	clause_keyword := "WHERE"
	if c.hasWhereAfterFrom {
		clause_keyword = "AND"
	} else {
		c.hasWhereAfterFrom = true
	}
	c.Text = strings.Replace(
		c.Text,
		"GROUP BY l.submitted_by",
		clause_keyword + " url LIKE ?\nGROUP BY l.submitted_by",
		1,
	)

	// Add arg in 2nd-to-last position before LIMIT
	last_arg := c.Args[len(c.Args) - 1]
	c.Args = c.Args[:len(c.Args) - 1]
	c.Args = append(c.Args, "%" + snippet + "%")
	c.Args = append(c.Args, last_arg)

	return c
}

func (c *Contributors) withURLLacking(snippet string) *Contributors {
	clause_keyword := "WHERE"
	if c.hasWhereAfterFrom {
		clause_keyword = "AND"
	} else {
		c.hasWhereAfterFrom = true
	}
	c.Text = strings.Replace(
		c.Text,
		"GROUP BY l.submitted_by",
		clause_keyword + " url NOT LIKE ?\nGROUP BY l.submitted_by",
		1,
	)

	// Add arg in 2nd-to-last position before LIMIT
	last_arg := c.Args[len(c.Args) - 1]
	c.Args = c.Args[:len(c.Args) - 1]
	c.Args = append(c.Args, "%" + snippet + "%")
	c.Args = append(c.Args, last_arg)

	return c
}

func (c *Contributors) duringPeriod(period string) *Contributors {
	if (period == "all") {
		return c
	}
	
	clause_keyword := "WHERE"
	if c.hasWhereAfterFrom {
		clause_keyword = "AND"
	}

	period_clause, err := getPeriodClause(period)
	if err != nil {
		c.Error = err
		return c
	}
	period_clause = strings.Replace(
		period_clause,
		"submit_date",
		"l.submit_date",
		1,
	)

	c.Text = strings.Replace(
		c.Text,
		"GROUP BY l.submitted_by",
		clause_keyword + " " + period_clause + "\n" + "GROUP BY l.submitted_by",
		1)

	return c
}
