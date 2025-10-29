package query

import (
	"fmt"
	"strings"

	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	mutil "github.com/julianlk522/modeep/model/util"
)

type TagRankingsForLink struct {
	*Query
}
type GlobalCatsForLink struct {
	*Query
}
type TopGlobalCatCounts struct {
	*Query
}
type SpellfixMatches struct {
	*Query

	// The key difference while submitting a link is that we don't use counts 
	// of subcats. When searching, it's useful to see only the cats that are 
	// included in the subset of links that you are potentially searching, so 
	// you can know how many results to expect. But when submitting a link you
	// should choose whatever cats you want whether or not there are existing
	// links with similar classifications. So for those, spellfix recommendations
	// just show the number of links that have them in their global cats 
	// ignoring already-applied filters but still excluding already-selected 
	// cats.)
	isNewLinkPage bool
}

// TAG RANKINGS FOR LINK
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

// GLOBAL CATS FOR LINK
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

// TOP GLOBAL CATS ACROSS ALL LINKS
func NewTopGlobalCatCounts() *TopGlobalCatCounts {
	return (&TopGlobalCatCounts{
		Query: &Query{
			Text: TOP_GLOBAL_CATS_BASE,
			Args: []any{GLOBAL_CATS_PAGE_LIMIT},
		},
	})
}

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
IndividualCatCounts AS (
    SELECT global_cat, count(DISTINCT id) as count
    FROM GlobalCatsSplit
    WHERE global_cat != ''
    GROUP BY LOWER(global_cat)
),
NormalizedCatCounts AS (
    SELECT 
        global_cat,
        count,
        CASE
            WHEN LOWER(global_cat) LIKE '%sses' THEN substr(LOWER(global_cat), 1, length(LOWER(global_cat)) - 2)
            WHEN LOWER(global_cat) LIKE '%s' AND NOT LOWER(global_cat) LIKE '%ss' THEN substr(LOWER(global_cat), 1, length(LOWER(global_cat)) - 1)
            ELSE LOWER(global_cat)
        END as normalized_global_cat
    FROM IndividualCatCounts
),
IdealSpellingVariants AS (
    SELECT 
        normalized_global_cat,
        SUM(count) as count_across_spelling_variations,
        (SELECT 
	    global_cat FROM NormalizedCatCounts ncc2 
	    WHERE ncc2.normalized_global_cat = ncc1.normalized_global_cat 
	    ORDER BY 
		count DESC, 
		length(global_cat) DESC, 
		global_cat DESC 
	    LIMIT 1
	) as ideal_cat_spelling
    FROM NormalizedCatCounts ncc1
    GROUP BY normalized_global_cat
)
SELECT 
    ideal_cat_spelling as global_cat, 
    count_across_spelling_variations as count
FROM IdealSpellingVariants
ORDER BY count_across_spelling_variations DESC
LIMIT ?;`

func (gcc *TopGlobalCatCounts) FromOptions(opts *model.TopCatCountsOptions) (*TopGlobalCatCounts, error) {
	if opts.RawCatFilters != nil {
		gcc = gcc.fromCatFilters(opts.RawCatFilters)
	}
	if opts.NeuteredCatFilters != nil {
		gcc = gcc.fromNeuteredCatFilters(opts.NeuteredCatFilters)
	}
	if opts.SummaryContains != "" {
		gcc = gcc.whereGlobalSummaryContains(opts.SummaryContains)
	}
	url_contains_params := opts.URLContains
	if url_contains_params != "" {
		gcc = gcc.whereURLContains(url_contains_params)
	}
	url_lacks_params := opts.URLLacks
	if url_lacks_params != "" {
		gcc = gcc.whereURLLacks(url_lacks_params)
	}
	period_params := opts.Period
	if period_params != "" {
		period := model.Period(period_params)
		if _, ok := model.ValidPeriodsInDays[period]; !ok {
			return nil, e.ErrInvalidPeriod
		} else {
			gcc = gcc.duringPeriod(period)
		}
	}
	more_params := opts.More
	if more_params {
		gcc = gcc.more()
	}
	if gcc.Error != nil {
		return nil, gcc.Error
	}
	return gcc, nil
}

func (gcc *TopGlobalCatCounts) fromCatFilters(raw_cat_filters []string) *TopGlobalCatCounts {
	if len(raw_cat_filters) == 0 {
		return gcc
	}

	// Build MATCH clause
	match_clause := `
	AND id IN (
		SELECT link_id
		FROM global_cats_fts
		WHERE global_cats MATCH ?
		)`

	// Build NOT IN clauses
	individual_cat_counts_not_in_clause := `
	AND LOWER(global_cat) NOT IN (?`
	normalized_cat_counts_not_in_clause := `
	WHERE normalized_global_cat NOT IN (?`

	for i := 1; i < len(raw_cat_filters); i++ {
		individual_cat_counts_not_in_clause += ", ?"
		normalized_cat_counts_not_in_clause += ", ?"
	}
	individual_cat_counts_not_in_clause += ")"
	normalized_cat_counts_not_in_clause += ")"

	// Add clauses
	gcc.Text = strings.Replace(
		gcc.Text,
		"WHERE global_cat != ''",
		"WHERE global_cat != ''" +
			match_clause +
			individual_cat_counts_not_in_clause,
		1,
	)
	gcc.Text = strings.Replace(
		gcc.Text,
		"FROM IndividualCatCounts",
		"FROM IndividualCatCounts" +
			normalized_cat_counts_not_in_clause,
		1,
	)

	// Build NOT IN args
	not_in_args := []any{strings.ToLower(raw_cat_filters[0])}
	for i := 1; i < len(raw_cat_filters); i++ {
		not_in_args = append(not_in_args, strings.ToLower(raw_cat_filters[i]))
	}

	// Build MATCH arg
	// cat_filters_With_spelling_variants := GetCatsOptionalPluralOrSingularForms(raw_cat_filters)
	// match_arg := cat_filters_With_spelling_variants[0]
	// for i := 1; i < len(cat_filters_With_spelling_variants); i++ {
	// 	match_arg += " AND " + cat_filters_With_spelling_variants[i]
	// }
	match_arg := strings.Join(
		GetCatsOptionalPluralOrSingularForms(raw_cat_filters),
		" AND ",
	)

	// Add args: {not_in_args...}, match_arg
	// old: [GLOBAL_CATS_PAGE_LIMIT]
	// new: [match_arg, not_in_args... x2, GLOBAL_CATS_PAGE_LIMIT]
	gcc.Args = append([]any{match_arg}, not_in_args...)
	gcc.Args = append(gcc.Args, not_in_args...)
	gcc.Args = append(gcc.Args, GLOBAL_CATS_PAGE_LIMIT)

	return gcc
}

func (gcc *TopGlobalCatCounts) fromNeuteredCatFilters(neutered_cat_filters []string) *TopGlobalCatCounts {
	if len(neutered_cat_filters) == 0 {
		return gcc
	}

	// Add CTEs
	gcc.Text = strings.Replace(
		gcc.Text,
		"WITH RECURSIVE ",
		"WITH\n" + GLOBAL_CAT_COUNTS_NEUTERED_CATS_CTES + ",\n",
		1,
	)

	// SELECT from new CTEs
	gcc.Text = strings.Replace(
		gcc.Text,
		"FROM Links",
		"FROM LinksWithNonNeuteredCats",
		1,
	)

	// Build MATCH arg
	// e.g., '("test" OR "tests") OR ("coding" OR "codings")'
	match_arg := strings.Join(
		GetCatsOptionalPluralOrSingularForms(neutered_cat_filters),
		" OR ",
	)

	// Add args
	// old: [GLOBAL_CATS_PAGE_LIMIT]
	// new: [match_arg, GLOBAL_CATS_PAGE_LIMIT]

	// OR if .fromCatFilters() called first:

	// old: [cat_filters_match_arg, cat_filters_not_in_args... x2,
	// GLOBAL_CATS_PAGE_LIMIT]
	// new: [match_arg, cat_filters_match_arg, cat_filters_not_in_args... x2,
	// GLOBAL_CATS_PAGE_LIMIT]
	// so can insert at the beginning
	gcc.Args = append([]any{match_arg}, gcc.Args...)
	return gcc
}

const GLOBAL_CAT_COUNTS_NEUTERED_CATS_CTES = `LinksWithNeuteredCats AS (
	SELECT link_id
	FROM global_cats_fts
	WHERE global_cats MATCH ?
),
LinksWithNonNeuteredCats AS (
	SELECT link_id as id, global_cats
	FROM global_cats_fts
	WHERE link_id NOT IN LinksWithNeuteredCats
)`

func (gcc *TopGlobalCatCounts) whereGlobalSummaryContains(snippet string) *TopGlobalCatCounts {
	// in case either .WhereURLContains or .WhereURLLacks was run first
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

func (gcc *TopGlobalCatCounts) whereURLContains(snippet string) *TopGlobalCatCounts {
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

func (gcc *TopGlobalCatCounts) whereURLLacks(snippet string) *TopGlobalCatCounts {
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

func (gcc *TopGlobalCatCounts) duringPeriod(period model.Period) *TopGlobalCatCounts {
	if period == "all" {
		return gcc
	}
	
	clause, err := getPeriodClause(period)
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

func (gcc *TopGlobalCatCounts) more() *TopGlobalCatCounts {
	gcc.Args[len(gcc.Args)-1] = MORE_GLOBAL_CATS_PAGE_LIMIT
	return gcc
}

// SPELLFIX
func NewSpellfixMatchesForSnippet(snippet string) *SpellfixMatches {
	return (&SpellfixMatches{
		Query: &Query{
			Text: "WITH " +
				SPELLFIX_MATCHES_CTE + ",\n" +
				NORMALIZED_MATCHES_CTE + ",\n" +
				COMBINED_RANKS_CTE + "\n" +
				SPELLFIX_SELECT,
			Args: []any{
				snippet,
				snippet + "*",
				SPELLFIX_MATCHES_LIMIT,
			},
		},
	})
}


// Rank represents total occurences across all global cats
// *including* potentially multiple times in the same link's global tag
// To prevent double-counting a cat with multiple variations inside the same link's
// global tag, cats are de-duped before having their spellfix ranks updated, rather
// than adjusting queries here to remove them.
const SPELLFIX_MATCHES_CTE = `SpellfixMatches AS (
    SELECT word, rank, distance
    FROM global_cats_spellfix gcs
    WHERE gcs.word MATCH ?
    OR gcs.word MATCH ?
    ORDER BY distance, rank DESC
)`

// e.g., "Tests" => "test"
const NORMALIZED_MATCHES_CTE = `NormalizedMatches AS (
	SELECT 
		word,
		rank,
		distance,
		CASE
			WHEN LOWER(word) LIKE '%sses' THEN substr(LOWER(word), 1, length(LOWER(word)) - 2)
			WHEN LOWER(word) LIKE '%s' AND NOT LOWER(word) LIKE '%ss' THEN substr(LOWER(word), 1, length(LOWER(word)) - 1)
			ELSE LOWER(word)
		END as normalized_word
	FROM SpellfixMatches
)`

// Combine ranks for variations of the same normalized form
// These would be merged into search results if the cat were selected as a filter,
// so the combined rank represents accurately how many links would appear.
const COMBINED_RANKS_CTE = `CombinedRanks AS (
	SELECT 
		normalized_word,
		SUM(rank) as combined_rank,
		MIN(distance) as min_distance,
		(SELECT word FROM NormalizedMatches nm2
		WHERE nm2.normalized_word = nm1.normalized_word
		ORDER BY rank DESC, length(word) DESC, word DESC
		LIMIT 1) as ideal_cat_spelling
	FROM NormalizedMatches nm1
	GROUP BY normalized_word
)`

// Sometimes computed distance comes out extremely low for unexpected results
// e.g., "co" => "computation" = 0, where "co" => coding = 100. wat
// Distance alone can't be trusted as a goodness-of-fit metric.
// To allow weighting tags through a combination of rank and distance,
// without having near-0 distances totally distort rankings, this lower bound
// is used.
const DISTANCE_LOWER_BOUND = 50
var SPELLFIX_SELECT = fmt.Sprintf(`SELECT 
	ideal_cat_spelling,
	combined_rank as rank
FROM CombinedRanks
ORDER BY (MAX(min_distance, %d) / combined_rank), combined_rank DESC
LIMIT ?;`, DISTANCE_LOWER_BOUND)

func (sm *SpellfixMatches) FromOptions(opts *model.SpellfixMatchesOptions) (*SpellfixMatches, error) {
	if opts.Tmap != "" {
		sm = sm.fromTmap(opts.Tmap)
	}
	if opts.IsNewLinkPage {
		sm.isNewLinkPage = true
	}
	if len(opts.CatFilters) > 0 {
		if sm.isNewLinkPage {
			sm = sm.fromCatFiltersWhileSubmittingLink(opts.CatFilters)
		} else {
			sm = sm.fromCatFilters(opts.CatFilters)
		}
	}
	if sm.Error != nil {
		return nil, sm.Error
	}
	return sm, nil
}

func (sm *SpellfixMatches) fromTmap(tmap_owner_login_name string) *SpellfixMatches {
	sm.Text = TMAP_SPELLFIX_BASE
		
	// Old: snippet, snippet*, LIMIT
	// New: tmap_owner_login_name, ("snippet" OR "snippets" OR "snippet"*) x2, tmap_owner_login_name x3, %snippet%, snippet, snippet*, LIMIT
	raw_snippet := sm.Args[0]
	snippet_with_spelling_variants := withOptionalPluralOrSingularForm(raw_snippet.(string))
	snippet_with_spelling_variants_and_wildcard := strings.Replace(
		snippet_with_spelling_variants,
		")",
		" OR " + getCatSurroundedInDoubleQuotes(raw_snippet.(string)) + "*)",
		1,
	)

	sm.Args = []any{
		tmap_owner_login_name,
		snippet_with_spelling_variants_and_wildcard ,
		snippet_with_spelling_variants_and_wildcard,
		tmap_owner_login_name,
		tmap_owner_login_name,
		tmap_owner_login_name,
		"%" + raw_snippet.(string) + "%",
		raw_snippet,
		raw_snippet.(string) + "*",
		SPELLFIX_MATCHES_LIMIT,
	}

	return sm
}

// For links in "Submitted" and "Starred" Treasure Map sections, we
// look primarily for user's cats and fall back to global cats for links which
// they did not tag. (If tagged then they submitted cats - use those.)
var TMAP_SPELLFIX_BASE = `WITH UserCatsMatches AS (
    SELECT 
        link_id,
        cats
    FROM user_cats_fts ucfts
    WHERE ucfts.submitted_by = ?
    AND cats MATCH ?
),
GlobalCatsMatches AS (
    SELECT
        link_id,
        global_cats as cats
    FROM global_cats_fts
    WHERE global_cats MATCH ?
    AND link_id IN (
		SELECT id FROM Links
		WHERE submitted_by = ?
		UNION 
		SELECT link_id FROM Stars
		WHERE user_id IN (
			SELECT id
			FROM Users
			WHERE login_name = ?
		)
    )
    AND link_id NOT IN (SELECT link_id FROM Tags WHERE submitted_by = ?)
),
CombinedCatsMatches AS (
    SELECT * FROM UserCatsMatches
    UNION ALL
    SELECT * FROM GlobalCatsMatches
),
CombinedCatsSplit(link_id, cat, remaining, cats) AS (
    SELECT 
        link_id,
        '',
        cats || ',',
        cats
    FROM CombinedCatsMatches
    UNION ALL 
    SELECT 
        link_id,
        SUBSTR(remaining, 0, INSTR(remaining, ',')),
        SUBSTR(remaining, INSTR(remaining, ',') + 1),
        cats
    FROM CombinedCatsSplit
    WHERE remaining != ''
),
MatchingCats AS (
    SELECT cat as word, link_id
    FROM CombinedCatsSplit
    WHERE cat != ''
    AND cat LIKE ?
    GROUP BY LOWER(cat), link_id
),
NormalizedMatches AS (
	SELECT 
		word,
		link_id,
		CASE
			WHEN LOWER(word) LIKE '%sses' THEN substr(LOWER(word), 1, length(LOWER(word)) - 2)
			WHEN LOWER(word) LIKE '%s' AND NOT LOWER(word) LIKE '%ss' THEN substr(LOWER(word), 1, length(LOWER(word)) - 1)
			ELSE LOWER(word)
		END as normalized_word
	FROM MatchingCats
	GROUP BY normalized_word, link_id
),
RankedNormalizedMatches AS (
	SELECT word, normalized_word, count(normalized_word) as rank_in_context
	FROM NormalizedMatches
	GROUP BY normalized_word
),
IdealSpellingVariants AS (
	SELECT word, normalized_word, count(word) as rank
	FROM NormalizedMatches
	GROUP BY word
	ORDER BY rank DESC
),
RankedIdealSpellingVariants AS (
	SELECT 
		rank_in_context,
		(SELECT word FROM IdealSpellingVariants isp
		WHERE isp.normalized_word = rnm.normalized_word
		ORDER BY isp.rank DESC, length(word) DESC, word DESC
		LIMIT 1) as ideal_cat_spelling
	FROM RankedNormalizedMatches rnm
),
SpellfixMatches AS (
    SELECT word, rank, distance
    FROM global_cats_spellfix gcs
    WHERE gcs.word MATCH ?
    OR gcs.word MATCH ?
    ORDER BY distance, rank DESC
),
FilteredSpellfixMatches AS (
	SELECT 
		sfm.word,
		(
			SELECT risv.rank_in_context
			FROM RankedIdealSpellingVariants risv
			WHERE risv.ideal_cat_spelling = sfm.word
		) as rank_in_context,
		sfm.distance
	FROM SpellfixMatches sfm
	WHERE rank_in_context IS NOT NULL
	
)` + "\n" + TMAP_SPELLFIX_SELECT

var TMAP_SPELLFIX_SELECT = fmt.Sprintf(`SELECT 
	word,
	rank_in_context as rank
FROM FilteredSpellfixMatches
ORDER BY (MAX(distance, %d) / rank_in_context), rank_in_context DESC
LIMIT ?;`, DISTANCE_LOWER_BOUND)

func (sm *SpellfixMatches) fromCatFilters(cat_filters []string) *SpellfixMatches {
	if len(cat_filters) == 0 || cat_filters[0] == "" {
		sm.Error = e.ErrNoCatFilters
		return sm
	}

	// Add placeholders to MatchingGlobalCats / MatchingCats CTEs NOT IN clause 
	// if more than 1 cat filter applied.
	// (MatchingGlobalCats or MatchingCats depending on if .FromTmap() called first)
	not_in_clause := "AND cat NOT IN (?"
	not_in_args := []any{}
	for i, cat := range cat_filters {
		if i > 0 {
			not_in_clause += ", ?"
		}
		not_in_args = append(not_in_args, cat)
	}
	not_in_clause += ")"

	// WHERE global_cats MATCH '("dog" OR "dogs") AND ("cat" OR "cats")', etc.
	fts_match_subcats_arg := strings.Join(GetCatsOptionalPluralOrSingularForms(cat_filters), " AND ")

	// Determine if .FromTmap() was called first
	// (likely a better way...)

	// .FromTmap() not called
	if len(sm.Args) == 3 {
		// Update CTES
		cat_filters_global_cats_ctes := CAT_FILTERS_GLOBAL_CATS_CTES
		if len(not_in_args) > 1 {
			cat_filters_global_cats_ctes = strings.Replace(
				cat_filters_global_cats_ctes,
				"AND cat NOT IN (?)",
				not_in_clause,
				1,
			)
		}
		sm.Text = strings.Replace(
			sm.Text,
			SPELLFIX_MATCHES_CTE,
			cat_filters_global_cats_ctes + ",\n" + SPELLFIX_MATCHES_CTE + ",\n" + FILTERED_SPELLFIX_MATCHES_CTE,
			1,
		)
		sm.Text = strings.Replace(
			sm.Text,
			NORMALIZED_MATCHES_CTE,
			FILTERED_NORMALIZED_MATCHES_CTE,
			1,
		)
		
		// Prepend args
		// Old: snippet, snippet*, LIMIT
		// New: fts_match_subcats_arg, not_in_args..., snippet, snippet*, LIMIT
		new_args := make([]any, 0, 1 + len(not_in_args) + len(sm.Args))
		new_args = append(new_args, fts_match_subcats_arg)
		new_args = append(new_args, not_in_args...)
		new_args = append(new_args, sm.Args...)
		sm.Args = new_args

	// .FromTmap() called first
	} else {
		// Update CTE
		new_matching_cats_cte := fmt.Sprintf(`MatchingCats AS (
			SELECT cat as word, link_id
			FROM CombinedCatsSplit
			WHERE cat != ''
			AND cat LIKE ?
			%s
		)`, not_in_clause)
		sm.Text = strings.Replace(
			sm.Text,
			`MatchingCats AS (
    SELECT cat as word, link_id
    FROM CombinedCatsSplit
    WHERE cat != ''
    AND cat LIKE ?
    GROUP BY LOWER(cat), link_id
)`,
			new_matching_cats_cte,
1,
		)

		// Insert args
		// Old: tmap_owner_login_name, ("snippet" OR "snippets" OR "snippet"*) x2, tmap_owner_login_name x3, %snippet%, snippet, snippet*, LIMIT
		// New: tmap_owner_login_name, ("snippet" OR "snippets" OR "snippet"* AND ("cat_filter" OR "cat_filters" ...)) x2, tmap_owner_login_name x3,
		// %snippet%, not_in_args..., snippet, snippet*, LIMIT

		// Add cat filters to UserCatsMatches / GlobalCatsMatches MATCH clauses to get subcats
		// Snippet with spelling variations are args[1] and args[2]
		old_match_arg := sm.Args[1]
		// "("test" OR "tests") => "("test" OR "tests") AND ("dog" OR "dogs") AND ("cat" OR "cats")
		new_match_arg := old_match_arg.(string) + " AND " + fts_match_subcats_arg
		sm.Args[1] = new_match_arg
		sm.Args[2] = new_match_arg

		// not_in_args... can be inserted 3 from the end
		up_to_last_3_args := sm.Args[:len(sm.Args) - 3]
		last_3_args := sm.Args[len(sm.Args) - 3:]
		
		new_args := make([]any, 0, len(sm.Args) + len(not_in_args))
		new_args = append(new_args, up_to_last_3_args...)
		new_args = append(new_args, not_in_args...)
		new_args = append(new_args, last_3_args...)
		sm.Args = new_args
	}

	return sm
}

var CAT_FILTERS_GLOBAL_CATS_CTES = `RECURSIVE MatchingGlobalCatsSplit(link_id, cat, remaining, global_cats) AS (
    SELECT 
        link_id,
        '',
        global_cats || ',',
        global_cats
    FROM global_cats_fts
	WHERE global_cats MATCH ?
    UNION ALL 
    SELECT 
        link_id,
        SUBSTR(remaining, 0, INSTR(remaining, ',')),
        SUBSTR(remaining, INSTR(remaining, ',') + 1),
        global_cats
    FROM MatchingGlobalCatsSplit
    WHERE remaining != ''
),
MatchingGlobalSubcats AS (
	SELECT cat, count(cat) as rank_in_context
	FROM MatchingGlobalCatsSplit
	WHERE cat != ''
	AND cat NOT IN (?)
	GROUP BY cat
)`

const FILTERED_SPELLFIX_MATCHES_CTE = `FilteredSpellfixMatches AS (
	SELECT 
		sfm.word,
		(
			SELECT mgs.rank_in_context 
			FROM MatchingGlobalSubcats mgs 
			WHERE mgs.cat = sfm.word) as rank_in_context,
		sfm.distance
	FROM SpellfixMatches sfm
	WHERE rank_in_context IS NOT NULL
)`

var FILTERED_NORMALIZED_MATCHES_CTE = `NormalizedMatches AS (
	SELECT 
		word,
		rank_in_context as rank,
		distance,
		CASE
			WHEN LOWER(word) LIKE '%sses' THEN substr(LOWER(word), 1, length(LOWER(word)) - 2)
			WHEN LOWER(word) LIKE '%s' AND NOT LOWER(word) LIKE '%ss' THEN substr(LOWER(word), 1, length(LOWER(word)) - 1)
			ELSE LOWER(word)
		END as normalized_word
	FROM FilteredSpellfixMatches
)`

func (sm *SpellfixMatches) fromCatFiltersWhileSubmittingLink(cat_filters []string) *SpellfixMatches {
	new_spellfix_matches_cte := SPELLFIX_MATCHES_FROM_CATS_WHILE_SUBMITTING_LINK_CTE
	if len(cat_filters) == 0 {
		return sm
	}
	var not_in_args = []any{cat_filters[0]}
	if len(cat_filters) > 1 {
		not_in_clause := "AND gcs.word NOT IN (?"
		for c:= 1; c < len(cat_filters); c++ {
			not_in_clause += ", ?"
			not_in_args = append(not_in_args, cat_filters[c])
		}
		not_in_clause += ")"
		new_spellfix_matches_cte = strings.Replace(
			new_spellfix_matches_cte,
			"AND gcs.word NOT IN (?)",
			not_in_clause,
			1,
		)
	}

	sm.Text = strings.Replace(
		sm.Text,
		SPELLFIX_MATCHES_CTE,
		new_spellfix_matches_cte,
		1,
	)

	// Insert args
	// Old: snippet, snippet*, LIMIT
	// New: snippet, snippet*, not_in_args..., LIMIT
	sm.Args = sm.Args[:len(sm.Args) - 1]
	sm.Args = append(sm.Args, not_in_args...)
	sm.Args = append(sm.Args, SPELLFIX_MATCHES_LIMIT)

	return sm
}

const SPELLFIX_MATCHES_FROM_CATS_WHILE_SUBMITTING_LINK_CTE = `SpellfixMatches AS (
    SELECT word, rank, distance
    FROM global_cats_spellfix gcs
    WHERE 
    	(gcs.word MATCH ? OR gcs.word MATCH ?)
    AND gcs.word NOT IN (?)
    ORDER BY distance, rank DESC
)`

