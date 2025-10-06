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
			Args: []any{link_id, NUM_TAGS_TO_SEARCH_FOR_TOP_CATS},
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

	tr.Args[1] = TAGS_PAGE_LIMIT

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
),
RawCatCounts AS (
    SELECT global_cat, count(DISTINCT id) as count
    FROM GlobalCatsSplit
    WHERE global_cat != ''
    GROUP BY LOWER(global_cat)
),
NormalizedGlobalCats AS (
    SELECT 
        global_cat,
        count,
        CASE
            WHEN LOWER(global_cat) LIKE '%sses' THEN substr(LOWER(global_cat), 1, length(LOWER(global_cat)) - 2)
            WHEN LOWER(global_cat) LIKE '%s' AND NOT LOWER(global_cat) LIKE '%ss' THEN substr(LOWER(global_cat), 1, length(LOWER(global_cat)) - 1)
            ELSE LOWER(global_cat)
        END as normalized_global_cat
    FROM RawCatCounts
),
IdealSpellingVariants AS (
    SELECT 
        normalized_global_cat,
        SUM(count) as max_count,
        (
			SELECT global_cat FROM NormalizedGlobalCats ngc2 
         	WHERE ngc2.normalized_global_cat = ngc1.normalized_global_cat 
         	ORDER BY count DESC, length(global_cat) DESC, global_cat DESC 
         	LIMIT 1
		) as ideal_cat_spelling
    FROM NormalizedGlobalCats ngc1
    GROUP BY normalized_global_cat
)
SELECT ideal_cat_spelling as global_cat, max_count as count
FROM IdealSpellingVariants
ORDER BY max_count DESC
LIMIT ?;`

func (gcc *GlobalCatCounts) FromRequestParams(params url.Values) *GlobalCatCounts {
	cats_params := params.Get("cats")
	if cats_params != "" {
		gcc = gcc.SubcatsOfCats(cats_params)
	}

	summary_contains_params := params.Get("summary_contains")
	if summary_contains_params != "" {
		gcc = gcc.WithGlobalSummaryContaining(summary_contains_params)
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
	// lowercase cats to ensure all case variations are returned
	cats := strings.Split(strings.ToLower(cats_params), ",")

	not_in_clause := `
	AND LOWER(global_cat) NOT IN (?`

	// add cat filter args to 2nd-to-last position
	// (LIMIT stays at the end - it is easier to add back after)
	limit_arg := gcc.Args[len(gcc.Args) - 1]
	gcc.Args = gcc.Args[:len(gcc.Args) - 1]

	gcc.Args = append(gcc.Args, cats[0])
	for i := 1; i < len(cats); i++ {
		not_in_clause += ", ?"
		gcc.Args = append(gcc.Args, cats[i])
	}
	not_in_clause += ")"

	// Add optional singular/plural variants
	// (skipped for NOT IN clause otherwise subcats include filters)
	match_arg := WithOptionalPluralOrSingularForm(cats[0])
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + WithOptionalPluralOrSingularForm(cats[i])
	}
	gcc.Args = append(gcc.Args, match_arg)

	match_clause := `
	AND id IN (
		SELECT link_id
		FROM global_cats_fts
		WHERE global_cats MATCH ?
		)`
	gcc.Text = strings.Replace(
		gcc.Text,
		"WHERE global_cat != ''",
		"WHERE global_cat != ''" +
			not_in_clause +
			match_clause,
		1,
	)

	gcc.Args = append(gcc.Args, limit_arg)

	return gcc
}

func (gcc *GlobalCatCounts) WithGlobalSummaryContaining(snippet string) *GlobalCatCounts {
	// in case either .WithURLContaining or .WithURLLacking was run first
	if strings.Contains(
		gcc.Text, 
		"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url)",
	) {
		gcc.Text = strings.Replace(
			gcc.Text,
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url)",
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url, global_summary)",
			1,
		)

		gcc.Text = strings.Replace(
			gcc.Text,
			"SELECT id, '', global_cats||',', url",
			"SELECT id, '', global_cats||',', url, global_summary",
			1,
		)

		gcc.Text = strings.Replace(
			gcc.Text,
			"substr(str, instr(str, ',') + 1),\nurl",
			"substr(str, instr(str, ',') + 1),\nurl,\nglobal_summary",
			1,
		)
	} else {
		gcc.Text = strings.Replace(
			gcc.Text,
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str)",
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, global_summary)",
			1,
		)

		gcc.Text = strings.Replace(
			gcc.Text,
			"SELECT id, '', global_cats||','",
			"SELECT id, '', global_cats||',', global_summary",
			1,
		)

		gcc.Text = strings.Replace(
			gcc.Text,
			"substr(str, instr(str, ',') + 1)",
			"substr(str, instr(str, ',') + 1),\nglobal_summary",
			1,
		)
	}

	gcc.Text = strings.Replace(
		gcc.Text,
		"WHERE str != ''",
		"WHERE str != ''\nAND global_summary LIKE ?",
		1,
	)

	// prepend arg
	gcc.Args = append([]any{"%" + snippet + "%"}, gcc.Args...)

	return gcc
}

func (gcc *GlobalCatCounts) WithURLContaining(snippet string) *GlobalCatCounts {
	// in case .WithGlobalSummaryContaining was run first
	if strings.Contains(
		gcc.Text, 
		"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, global_summary)",
	) {
		gcc.Text = strings.Replace(
			gcc.Text,
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, global_summary)",
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url, global_summary)",
			1,
		)

		gcc.Text = strings.Replace(
			gcc.Text,
			"SELECT id, '', global_cats||',', global_summary",
			"SELECT id, '', global_cats||',', url, global_summary",
			1,
		)

		gcc.Text = strings.Replace(
			gcc.Text,
			"substr(str, instr(str, ',') + 1),\nglobal_summary",
			"substr(str, instr(str, ',') + 1),\nurl,\nglobal_summary",
			1,
		)
	} else if !strings.Contains(
		gcc.Text,
		"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url)",
	) {
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
		"WHERE str != ''",
		"WHERE str != ''\nAND url LIKE ?",
		1,
	)

	// prepend arg
	gcc.Args = append([]any{"%" + snippet + "%"}, gcc.Args...)

	return gcc
}

func (gcc *GlobalCatCounts) WithURLLacking(snippet string) *GlobalCatCounts {
	// in case .WithGlobalSummaryContaining was run first
	if strings.Contains(
		gcc.Text, 
		"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, global_summary)",
	) {
		gcc.Text = strings.Replace(
			gcc.Text,
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, global_summary)",
			"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url, global_summary)",
			1,
		)

		gcc.Text = strings.Replace(
			gcc.Text,
			"SELECT id, '', global_cats||',', global_summary",
			"SELECT id, '', global_cats||',', url, global_summary",
			1,
		)

		gcc.Text = strings.Replace(
			gcc.Text,
			"substr(str, instr(str, ',') + 1),\nglobal_summary",
			"substr(str, instr(str, ',') + 1),\nurl,\nglobal_summary",
			1,
		)
	} else if !strings.Contains(
		gcc.Text,
		"WITH RECURSIVE GlobalCatsSplit(id, global_cat, str, url",
	) {
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
		"WHERE str != ''",
		"WHERE str != ''\nAND url NOT LIKE ?",
		1,
	)

	// prepend arg
	gcc.Args = append([]any{"%" + snippet + "%"}, gcc.Args...)

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
