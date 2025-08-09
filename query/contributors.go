package query

import (
	"net/url"
	"strings"
)

const CONTRIBUTORS_PAGE_LIMIT = 10

type Contributors struct {
	*Query
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
	cats_params := params.Get("cats")
	if cats_params != "" {
		cats := strings.Split(cats_params, ",")
		c = c.FromCats(cats)
	}

	url_contains_params := params.Get("url_contains")
	if url_contains_params != "" {
		c = c.WithURLContaining(url_contains_params)
	}

	url_lacks_params := params.Get("url_lacks")
	if url_lacks_params != "" {
		c = c.WithURLLacking(url_lacks_params)
	}

	period_params := params.Get("period")
	if period_params != "" {
		c = c.DuringPeriod(period_params)
	}

	return c
}

func (c *Contributors) FromCats(cats []string) *Contributors {
	if len(cats) == 0 {
		return c
	}

	cats = GetCatsOptionalPluralOrSingularForms(cats)
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}
	c.Args = append(c.Args, match_arg)

	// Build CTE
	match_clause := " WHERE global_cats MATCH ?"
	cats_CTE := `WITH CatsFilter AS (
	SELECT link_id
	FROM global_cats_fts` + match_clause + `
	)
	`

	// Prepend CTE
	c.Text = cats_CTE + c.Text

	// Append join
	c.Text = strings.Replace(
		c.Text,
		"FROM Links l",
		"FROM Links l"+CONTRIBUTORS_CATS_FROM,
		1,
	)

	// Move page limit arg from first to last
	c.Args = append(c.Args[1:], c.Args[0])

	return c
}

const CONTRIBUTORS_CATS_FROM = `
INNER JOIN CatsFilter f ON l.id = f.link_id`

func (c *Contributors) WithURLContaining(snippet string) *Contributors {
	c.Text = strings.Replace(
		c.Text,
		"GROUP BY l.submitted_by",
		"WHERE url LIKE ?\nGROUP BY l.submitted_by",
		1,
	)

	// insert into arg in 2nd-to-last position
	last_arg := c.Args[len(c.Args)-1]
	c.Args = c.Args[:len(c.Args)-1]
	c.Args = append(c.Args, "%"+snippet+"%")
	c.Args = append(c.Args, last_arg)

	return c
}

func (c *Contributors) WithURLLacking(snippet string) *Contributors {
	c.Text = strings.Replace(
		c.Text,
		"GROUP BY l.submitted_by",
		"WHERE url NOT LIKE ?\nGROUP BY l.submitted_by",
		1,
	)

	// insert into arg in 2nd-to-last position
	last_arg := c.Args[len(c.Args)-1]
	c.Args = c.Args[:len(c.Args)-1]
	c.Args = append(c.Args, "%"+snippet+"%")
	c.Args = append(c.Args, last_arg)

	return c
}

func (c *Contributors) DuringPeriod(period string) *Contributors {
	if (period == "all") {
		return c
	}
	
	var clause_keyword string
	// adapt keyword if .WithURLContaining was called first
	if strings.Contains(c.Text, "WHERE url LIKE") {
		clause_keyword = "AND"
	} else {
		clause_keyword = "WHERE"
	}

	period_clause, err := GetPeriodClause(period)
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
		clause_keyword+" "+period_clause+"\n"+"GROUP BY l.submitted_by",
		1)

	return c
}
