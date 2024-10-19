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
			Text: 
				CONTRIBUTORS_BASE,
			Args: 
				[]interface{}{CONTRIBUTORS_PAGE_LIMIT},
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

	EscapeCatsReservedChars(cats)

	clause := " WHERE global_cats MATCH ?"
	match_arg := cats[0]
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + cats[i]
	}
	c.Args = append(c.Args, match_arg)

	// build CTE
	cats_cte := `WITH CatsFilter AS (
	SELECT link_id
	FROM global_cats_fts` + clause + `
	)
	`

	// prepend CTE
	c.Text = cats_cte + c.Text

	// append join
	c.Text = strings.Replace(
		c.Text,
		"FROM Links l",
		"FROM Links l" + CONTRIBUTORS_CATS_FROM,
		1,
	)

	// move page limit arg from first to last
	c.Args = append(c.Args[1:], c.Args[0])

	return c
}

const CONTRIBUTORS_CATS_FROM = `
INNER JOIN CatsFilter f ON l.id = f.link_id`

func (c *Contributors) DuringPeriod(period string) *Contributors {
	clause, err := GetPeriodClause(period)
	if err != nil {
		c.Error = err
		return c
	}

	clause = strings.Replace(
		clause,
		"submit_date",
		"l.submit_date",
		1,
	)

	// Prepend new clause
	c.Text = strings.Replace(
		c.Text,
		"GROUP BY l.submitted_by",
		"WHERE "+clause+"\n"+"GROUP BY l.submitted_by",
		1)

	return c
}