package query

import (
	"fmt"
	"net/url"
	"strings"

	e "github.com/julianlk522/modeep/error"
)

type TagRankings struct {
	*Query
}

func NewTagRankings(link_id string) *TagRankings {
	return (&TagRankings{Query: 
		&Query{
			Text: TAG_RANKINGS_BASE,
			Args: []any{link_id, TAG_RANKINGS_CALC_LIMIT},
		},
	})
}

const TAG_RANKINGS_BASE_FIELDS = `SELECT
	(julianday('now') - julianday(last_updated)) / (julianday('now') - julianday(submit_date)) * 100 AS lifespan_overlap, 
	cats`

var TAG_RANKINGS_BASE = TAG_RANKINGS_BASE_FIELDS + ` 
FROM Tags 
INNER JOIN Links 
ON Links.id = Tags.link_id
WHERE link_id = ?
ORDER BY lifespan_overlap DESC
LIMIT ?;`

func (tr *TagRankings) Public() *TagRankings {
	tr.Text = strings.Replace(
		tr.Text,
		TAG_RANKINGS_BASE_FIELDS,
		TAG_RANKINGS_BASE_FIELDS+TAG_RANKINGS_PUBLIC_FIELDS,
		1,
	)

	tr.Args[1] = TAG_RANKINGS_PAGE_LIMIT

	return tr
}

const TAG_RANKINGS_PUBLIC_FIELDS = `, 
	Tags.submitted_by, 
	last_updated`

type GlobalCatCounts struct {
	*Query
}

func NewTopGlobalCatCounts() *GlobalCatCounts {
	return (&GlobalCatCounts{
		Query: &Query{
			Text: GLOBAL_CATS_BASE,
			Args: []any{GLOBAL_CATS_PAGE_LIMIT},
		},
	})
}

// id used for .SubcatsOfCats(): don't remove
const GLOBAL_CATS_BASE = `WITH RECURSIVE GlobalCatsSplit(id, global_cat, str) AS (
    SELECT id, '', global_cats||','
    FROM Links
    UNION ALL SELECT
	id,
    substr(str, 0, instr(str, ',')),
    substr(str, instr(str, ',') + 1)
    FROM GlobalCatsSplit
    WHERE str != ''
)
SELECT global_cat, count(DISTINCT id) as count
FROM GlobalCatsSplit
WHERE global_cat != ''
GROUP BY LOWER(global_cat)
ORDER BY count DESC, LOWER(global_cat) ASC
LIMIT ?`

func (gcc *GlobalCatCounts) FromRequestParams(params url.Values) *GlobalCatCounts {
	cats_params := params.Get("cats")
	if cats_params != "" {
		gcc = gcc.SubcatsOfCats(cats_params)
	}

	url_contains_params := params.Get("url_contains")
	if url_contains_params != "" {
		gcc = gcc.WithURLContaining(url_contains_params)
	}

	url_lacks_params := params.Get("url_lacks")
	if url_lacks_params != "" {
		gcc = gcc.WithURLLacking(url_lacks_params)
	}

	period_params := params.Get("period")
	if period_params != "" {
		gcc = gcc.DuringPeriod(period_params)
	}

	more_params := params.Get("more")
	if more_params == "true" {
		gcc = gcc.More()
	} else if more_params != "" {
		gcc.Error = e.ErrInvalidMoreFlag
	}

	return gcc
}

func (gcc *GlobalCatCounts) SubcatsOfCats(cats_params string) *GlobalCatCounts {
	// Lowercase to ensure all case variations are returned
	cats := strings.Split(strings.ToLower(cats_params), ",")

	// Build NOT IN clause
	not_in_clause := `
	AND LOWER(global_cat) NOT IN (?`

	gcc.Args = append(gcc.Args, cats[0])
	for i := 1; i < len(cats); i++ {
		not_in_clause += ", ?"
		gcc.Args = append(gcc.Args, cats[i])
	}
	not_in_clause += ")"

	// Build MATCH clause
	match_clause := `
	AND id IN (
		SELECT link_id
		FROM global_cats_fts
		WHERE global_cats MATCH ?
		)`

	match_arg := WithOptionalPluralOrSingularForm(cats[0])
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + WithOptionalPluralOrSingularForm(cats[i])
	}
	// Add optional singular/plural variants
	// (skip for NOT IN clause otherwise subcats include filters)
	gcc.Args = append(gcc.Args, match_arg)

	gcc.Text = strings.Replace(
		gcc.Text,
		"WHERE global_cat != ''",
		"WHERE global_cat != ''"+
			not_in_clause+
			match_clause,
		1)

	// Move LIMIT arg to end
	gcc.Args = append(gcc.Args[1:], GLOBAL_CATS_PAGE_LIMIT)

	return gcc
}

func (gcc *GlobalCatCounts) WithURLContaining(snippet string) *GlobalCatCounts {
	gcc.Text = strings.Replace(
		gcc.Text,
		"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str)",
		"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url)",
		1,
	)
	gcc.Text = strings.Replace(
		gcc.Text,
		"SELECT id, '', global_cats||','",
		"SELECT id, '', global_cats||',', url",
		1,
	)
	gcc.Text = strings.Replace(
		gcc.Text,
		"substr(str, instr(str, ',') + 1)",
		"substr(str, instr(str, ',') + 1),\nurl",
		1,
	)
	gcc.Text = strings.Replace(
		gcc.Text,
		"GROUP BY LOWER(global_cat)",
		"\nAND url LIKE ?\nGROUP BY LOWER(global_cat)",
		1,
	)

	// insert into args in 2nd-to-last position
	last_arg := gcc.Args[len(gcc.Args)-1]
	gcc.Args = gcc.Args[:len(gcc.Args)-1]
	gcc.Args = append(gcc.Args, "%"+snippet+"%")
	gcc.Args = append(gcc.Args, last_arg)

	return gcc
}

func (gcc *GlobalCatCounts) WithURLLacking(snippet string) *GlobalCatCounts {

	// check if WithURLContaining was already run
	// (avoid double string replacement)
	has_url_contains := strings.Contains(
		gcc.Text,
		"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url)",
	) &&
		strings.Contains(
		gcc.Text,
		"SELECT id, '', global_cats||',', url",
	) &&
		strings.Contains(
		gcc.Text,
		"substr(str, instr(str, ',') + 1),\nurl",
	)

	if !has_url_contains {
		gcc.Text = strings.Replace(
			gcc.Text,
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str)",
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url)",
			1,
		)
		gcc.Text = strings.Replace(
			gcc.Text,
			"SELECT id, '', global_cats||','",
			"SELECT id, '', global_cats||',', url",
			1,
		)
		gcc.Text = strings.Replace(
			gcc.Text,
			"substr(str, instr(str, ',') + 1)",
			"substr(str, instr(str, ',') + 1),\nurl",
			1,
		)
	}
	
	gcc.Text = strings.Replace(
		gcc.Text,
		"GROUP BY LOWER(global_cat)",
		"AND url NOT LIKE ?\nGROUP BY LOWER(global_cat)",
		1,
	)

	// insert into args in 2nd-to-last position
	last_arg := gcc.Args[len(gcc.Args)-1]
	gcc.Args = gcc.Args[:len(gcc.Args)-1]
	gcc.Args = append(gcc.Args, "%"+snippet+"%")
	gcc.Args = append(gcc.Args, last_arg)

	return gcc
}

func (gcc *GlobalCatCounts) DuringPeriod(period string) *GlobalCatCounts {
	if period == "all" {
		return gcc
	}
	
	clause, err := GetPeriodClause(period)
	if err != nil {
		gcc.Error = err
		return gcc
	}

	gcc.Text = strings.Replace(
		gcc.Text,
		"FROM Links",
		fmt.Sprintf(
			`FROM Links
			WHERE %s`,
			clause,
		),
		1)

	return gcc
}

func (gcc *GlobalCatCounts) More() *GlobalCatCounts {
	gcc.Args[len(gcc.Args)-1] = MORE_GLOBAL_CATS_PAGE_LIMIT
	return gcc
}

type SpellfixMatches struct {
	*Query
}

func NewSpellfixMatchesForSnippet(snippet string) *SpellfixMatches {
	// oddly, "WHERE word MATCH "%s OR %s*" doesn't work very well here
	// hence the UNION
	return (&SpellfixMatches{
		Query: &Query{
			Text: `WITH CombinedResults AS (
		SELECT word, rank, distance
		FROM global_cats_spellfix
		WHERE word MATCH ?
		UNION ALL
		SELECT word, rank, distance
		FROM global_cats_spellfix
		WHERE word MATCH ? || '*'
			),
		RankedResults AS (
			SELECT 
				word, 
				rank,
				distance,
				ROW_NUMBER() OVER (PARTITION BY word ORDER BY distance) AS row_num
			FROM CombinedResults
		),
		TopResults AS (
			SELECT word, rank, distance
			FROM RankedResults
			WHERE row_num = 1
			AND distance <= ?
			ORDER BY distance, rank DESC
		)
		SELECT word, SUM(rank) as rank
		FROM TopResults
		GROUP BY LOWER(word)
		ORDER BY distance, rank DESC
		LIMIT ?;`,
			Args: []any{
				snippet,
				snippet,
				SPELLFIX_DISTANCE_LIMIT,
				SPELLFIX_MATCHES_LIMIT,
			},
		},
	})
}

func (sm *SpellfixMatches) OmitCats(cats []string) error {
	if len(cats) == 0 || cats[0] == "" {
		return e.ErrNoOmittedCats
	}

	// Pop SPELLFIX_MATCHES_LIMIT arg
	sm.Args = sm.Args[0 : len(sm.Args)-1]

	not_in_clause := `
	AND LOWER(word) NOT IN (?`
	sm.Args = append(sm.Args, cats[0])

	for i := 1; i < len(cats); i++ {
		not_in_clause += ", ?"
		sm.Args = append(sm.Args, cats[i])
	}
	not_in_clause += ")"

	distance_clause := `AND distance <= ?`
	sm.Text = strings.Replace(
		sm.Text,
		distance_clause,
		distance_clause+not_in_clause,
		1,
	)

	// Push SPELLFIX_MATCHES_LIMIT arg back to end
	sm.Args = append(sm.Args, SPELLFIX_MATCHES_LIMIT)

	return nil
}
