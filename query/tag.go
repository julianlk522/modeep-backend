package query

import (
	"fmt"
	"net/url"
	"strings"

	e "github.com/julianlk522/modeep/error"
	mutil "github.com/julianlk522/modeep/model/util"
)

type TagRankingsForLink struct {
	*Query
}

func NewTagRankingsForLink(link_id string) *TagRankingsForLink {
	return (&TagRankingsForLink{
		Query: &Query{
			Text: `SELECT
	(julianday('now') - julianday(t.last_updated)) / (julianday('now') - julianday(l.submit_date)) * 100 AS lifespan_overlap, 
	t.cats, 
	t.submitted_by, 
	t.last_updated
FROM Tags t
INNER JOIN Links l
ON l.id = t.link_id
WHERE t.link_id = ?
ORDER BY lifespan_overlap DESC
LIMIT ?;`,
			Args: []any{
				link_id, 
				TAGS_PAGE_LIMIT,
			},
		},
	})
}

type GlobalCatsForLink struct {
	*Query
}

func NewGlobalCatsForLink(link_id string) *GlobalCatsForLink {
	return (&GlobalCatsForLink{
		Query: &Query{
			Text: `WITH TagLifespanOverlaps AS (
    SELECT
        (julianday('now') - julianday(t.last_updated)) / (julianday('now') - julianday(l.submit_date)) * 100 AS lifespan_overlap,
        t.link_id,
        t.cats as cats
    FROM Tags t
    INNER JOIN Links l ON l.id = t.link_id
    WHERE t.link_id = ?
    ORDER BY lifespan_overlap DESC
    LIMIT ?
),
CatsSplit(link_id, lifespan_overlap, cat, str) AS (
    SELECT link_id, lifespan_overlap, '', cats||','
    FROM TagLifespanOverlaps
    UNION ALL 
    SELECT
        link_id,
	lifespan_overlap,
        TRIM(substr(str, 0, instr(str, ','))),
        substr(str, instr(str, ',') + 1)
    FROM CatsSplit
    WHERE str != ''
),
IndividualCats AS (
    SELECT DISTINCT
        link_id,
	lifespan_overlap,
        cat
    FROM CatsSplit
    WHERE cat != ''
),
NormalizedCats AS (
    SELECT 
        link_id,
	lifespan_overlap,
        cat,
        CASE
            WHEN LOWER(cat) LIKE '%sses' THEN substr(LOWER(cat), 1, length(LOWER(cat)) - 2)
            WHEN LOWER(cat) LIKE '%s' AND NOT LOWER(cat) LIKE '%ss' THEN substr(LOWER(cat), 1, length(LOWER(cat)) - 1)
            ELSE LOWER(cat)
        END as normalized_cat
    FROM IndividualCats
),
IdealSpellingVariants AS (
    SELECT 
        link_id,
        normalized_cat,
	SUM(lifespan_overlap) as cat_score,
        (SELECT 
	    cat FROM NormalizedCats nc2 
	    WHERE nc2.link_id = nc1.link_id 
	    AND nc2.normalized_cat = nc1.normalized_cat
	    ORDER BY length(cat) DESC, cat DESC 
	    LIMIT 1
        ) as ideal_cat_spelling
    FROM NormalizedCats nc1
    GROUP BY normalized_cat
    ORDER BY 
	cat_score DESC, 
	length(ideal_cat_spelling) ASC, 
	ideal_cat_spelling ASC
    LIMIT ?
),
MaxScore AS (
    SELECT MAX(cat_score) as high_score
    FROM IdealSpellingVariants
)
SELECT GROUP_CONCAT(ideal_cat_spelling, ',') as new_global_cats
FROM IdealSpellingVariants, MaxScore
WHERE cat_score >= high_score / 100 * ?;`,
			Args: []any{
				link_id, 
				TAGS_TO_SEARCH_FOR_TOP_GLOBAL_CATS,
				mutil.CATS_PER_LINK_LIMIT,
				PERCENT_OF_MAX_CAT_SCORE_NEEDED_FOR_GLOBAL_CATS_ASSIGNMENT,
			},
		},
	})
}

type TopGlobalCatCounts struct {
	*Query
}

func NewTopGlobalCatCounts() *TopGlobalCatCounts {
	return (&TopGlobalCatCounts{
		Query: &Query{
			Text: TOP_GLOBAL_CATS_BASE,
			Args: []any{GLOBAL_CATS_PAGE_LIMIT},
		},
	})
}

// id used for .SubcatsOfCats(): don't remove
const TOP_GLOBAL_CATS_BASE = `WITH RECURSIVE GlobalCatsSplit(id, global_cat, str) AS (
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
        SUM(count) as count_across_spelling_variations,
        (SELECT 
	    global_cat FROM NormalizedGlobalCats ngc2 
	    WHERE ngc2.normalized_global_cat = ngc1.normalized_global_cat 
	    ORDER BY 
		count DESC, 
		length(global_cat) DESC, 
		global_cat DESC 
	    LIMIT 1
	) as ideal_cat_spelling
    FROM NormalizedGlobalCats ngc1
    GROUP BY normalized_global_cat
)
SELECT 
    ideal_cat_spelling as global_cat, 
    count_across_spelling_variations as count
FROM IdealSpellingVariants
ORDER BY count_across_spelling_variations DESC
LIMIT ?;`

func (gcc *TopGlobalCatCounts) FromRequestParams(params url.Values) *TopGlobalCatCounts {
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

func (gcc *TopGlobalCatCounts) SubcatsOfCats(cats_params string) *TopGlobalCatCounts {
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

func (gcc *TopGlobalCatCounts) WithGlobalSummaryContaining(snippet string) *TopGlobalCatCounts {
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

func (gcc *TopGlobalCatCounts) WithURLContaining(snippet string) *TopGlobalCatCounts {
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

func (gcc *TopGlobalCatCounts) WithURLLacking(snippet string) *TopGlobalCatCounts {
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

func (gcc *TopGlobalCatCounts) DuringPeriod(period string) *TopGlobalCatCounts {
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

func (gcc *TopGlobalCatCounts) More() *TopGlobalCatCounts {
	gcc.Args[len(gcc.Args)-1] = MORE_GLOBAL_CATS_PAGE_LIMIT
	return gcc
}

type SpellfixMatches struct {
	*Query
}

func NewSpellfixMatchesForSnippet(snippet string) *SpellfixMatches {
	return (&SpellfixMatches{
		Query: &Query{
			Text: "WITH " +
				SPELLFIX_BASE_CTE +
				REST_OF_SPELLFIX_BASE_QUERY,
			Args: []any{
				snippet,
				snippet,
				SPELLFIX_DISTANCE_LIMIT,
				SPELLFIX_MATCHES_LIMIT,
			},
		},
	})
}

func (sm *SpellfixMatches) FromTmap(tmap_owner_login_name string) *SpellfixMatches {
	sm.Text = strings.Replace(
		sm.Text,
		SPELLFIX_BASE_CTE,
		`RECURSIVE CatsSplit(tag_id, link_id, submitted_by, cat, remaining) AS (
	SELECT 
		tag_id, 
		link_id, 
		submitted_by,
		'',
		cats || ','
	FROM user_cats_fts
	WHERE submitted_by = ?
	UNION ALL 
	SELECT 
		tag_id,
		link_id,
		submitted_by,
		SUBSTR(remaining, 0, INSTR(remaining, ',')),
		SUBSTR(remaining, INSTR(remaining, ',') + 1)
	FROM CatsSplit
	WHERE remaining != ''
),
CatRanks AS (
	SELECT cat, COUNT(*) as local_rank
	FROM CatsSplit
	WHERE cat != ''
	GROUP BY cat
),
SpellfixMatches AS (
	SELECT gcs.word, cr.local_rank, gcs.distance
	FROM global_cats_spellfix gcs
	INNER JOIN CatRanks cr ON LOWER(gcs.word) = LOWER(cr.cat)
	WHERE gcs.word MATCH ?
	UNION ALL
	SELECT gcs.word, cr.local_rank, gcs.distance
	FROM global_cats_spellfix gcs
	INNER JOIN CatRanks cr ON LOWER(gcs.word) = LOWER(cr.cat)
	WHERE gcs.word MATCH ? || '*'
)`,
		1,
	)

	// this is not strictly necessary - could use "rank" as in the base query,
	// but this is a bit more clear.
	new_rest_of_query := strings.ReplaceAll(
		REST_OF_SPELLFIX_BASE_QUERY,
		"rank",
		"local_rank",
	)
	
	sm.Text = strings.Replace(
		sm.Text,
		REST_OF_SPELLFIX_BASE_QUERY,
		new_rest_of_query,
		1,
	)

	// prepend arg
	sm.Args = append([]any{tmap_owner_login_name}, sm.Args...)

	return sm
}

// oddly, "WHERE word MATCH "%s OR %s*" doesn't work very well
// hence the UNION
const SPELLFIX_BASE_CTE = `SpellfixMatches AS (
	SELECT word, rank, distance
	FROM global_cats_spellfix
	WHERE word MATCH ?
	UNION ALL
	SELECT word, rank, distance
	FROM global_cats_spellfix
	WHERE word MATCH ? || '*'
)`

const REST_OF_SPELLFIX_BASE_QUERY = `,
RankedMatches AS (
	SELECT 
		word, 
		rank,
		distance,
		ROW_NUMBER() OVER (PARTITION BY word ORDER BY distance, rank DESC) AS row_num
	FROM SpellfixMatches
),
TopMatches AS (
	SELECT 
		word, 
		rank, 
		distance
	FROM RankedMatches
	WHERE row_num = 1
	AND distance <= ?
	ORDER BY distance, rank DESC
)
SELECT word, SUM(rank) as rank
FROM TopMatches
GROUP BY LOWER(word)
ORDER BY distance, rank DESC
LIMIT ?;`

func (sm *SpellfixMatches) OmitCats(cats []string) error {
	if len(cats) == 0 || cats[0] == "" {
		return e.ErrNoOmittedCats
	}

	// Pop SPELLFIX_MATCHES_LIMIT arg
	sm.Args = sm.Args[0 : len(sm.Args) - 1]

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
		distance_clause + not_in_clause,
		1,
	)

	// Push SPELLFIX_MATCHES_LIMIT arg back to end
	sm.Args = append(sm.Args, SPELLFIX_MATCHES_LIMIT)

	return nil
}
