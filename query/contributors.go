package query

import (
	"strings"
)

const CONTRIBUTORS_PAGE_LIMIT = 10

type Contributors struct {
	*Query
}

func NewContributors() *Contributors {
	return (&Contributors{
		Query: &Query{
			Text: CONTRIBUTORS_BASE,
			Args: []interface{}{CONTRIBUTORS_PAGE_LIMIT},
		},
	})
}

const CONTRIBUTORS_BASE = `SELECT
count(l.id) as count, l.submitted_by
FROM Links l
GROUP BY l.submitted_by
ORDER BY count DESC, l.submitted_by ASC
LIMIT ?;`

func (c *Contributors) FromCats(cats []string) *Contributors {
	if len(cats) == 0 {
		return c
	}

	cats = GetCatsOptionalPluralOrSingularForms(
		GetCatsWithEscapedReservedChars(cats),
	)

	match_clause := " WHERE global_cats MATCH ?"
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}
	c.Args = append(c.Args, match_arg)

	// Build CTE
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

func (c *Contributors) DuringPeriod(period string) *Contributors {
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
		"WHERE "+period_clause+"\n"+"GROUP BY l.submitted_by",
		1)

	return c
}
