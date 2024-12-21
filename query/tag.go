package query

import (
	"fmt"
	"strings"

	e "github.com/julianlk522/fitm/error"
)

const (
	TAG_RANKINGS_PAGE_LIMIT     = 20
	GLOBAL_CATS_PAGE_LIMIT      = 20
	MORE_GLOBAL_CATS_PAGE_LIMIT = 100

	SPELLFIX_DISTANCE_LIMIT = 100
	SPELLFIX_MATCHES_LIMIT  = 3
)

type TagPageLink struct {
	*Query
}

func NewTagPageLink(link_id string) *TagPageLink {
	return (&TagPageLink{Query: &Query{
		Text: TAG_PAGE_LINK_BASE_FIELDS +
			TAG_PAGE_LINK_BASE_FROM +
			TAG_PAGE_LINK_BASE_JOINS,
		Args: []interface{}{link_id},
	},
	})
}

const TAG_PAGE_LINK_BASE_FIELDS = `SELECT 
	links_id as link_id, 
	url, 
	sb, 
	sd, 
	cats, 
	summary,
	COALESCE(summary_count,0) as summary_count, 
	COUNT("Link Likes".id) as like_count, 
	img_url`

const TAG_PAGE_LINK_BASE_FROM = `
FROM 
	(
	SELECT 
		id as links_id, 
		url, 
		submitted_by as sb, 
		submit_date as sd, 
		COALESCE(global_cats,"") as cats, 
		global_summary as summary, 
		COALESCE(img_url,"") as img_url 
	FROM Links
	WHERE id = ?
	)`

const TAG_PAGE_LINK_BASE_JOINS = `
LEFT JOIN "Link Likes"
ON "Link Likes".link_id = links_id
LEFT JOIN
	(
	SELECT count(*) as summary_count, link_id as slink_id
	FROM Summaries
	GROUP BY slink_id
	)
ON slink_id = links_id;`

func (tpl *TagPageLink) AsSignedInUser(user_id string) *TagPageLink {
	tpl.Text = strings.Replace(
		tpl.Text,
		TAG_PAGE_LINK_BASE_FIELDS,
		TAG_PAGE_LINK_BASE_FIELDS+TAG_PAGE_LINK_AUTH_FIELDS,
		1,
	)

	tpl.Text = strings.Replace(
		tpl.Text,
		";",
		TAG_PAGE_LINK_AUTH_JOINS,
		1,
	)
	tpl.Args = append(tpl.Args, user_id, user_id)

	return tpl
}

const TAG_PAGE_LINK_AUTH_FIELDS = `,
	COALESCE(is_liked,0) as is_liked, 
	COALESCE(is_copied,0) as is_copied`

const TAG_PAGE_LINK_AUTH_JOINS = ` 
	LEFT JOIN
		(
		SELECT id as like_id, count(*) as is_liked, user_id as luser_id, link_id as like_link_id2
		FROM "Link Likes"
		WHERE luser_id = ?
		GROUP BY like_id
		)
	ON like_link_id2 = link_id
	LEFT JOIN
		(
		SELECT id as copy_id, count(*) as is_copied, user_id as cuser_id, link_id as copy_link_id
		FROM "Link Copies"
		WHERE cuser_id = ?
		GROUP BY copy_id
		)
	ON copy_link_id = link_id;`

type TagRankings struct {
	*Query
}

func NewTagRankings(link_id string) *TagRankings {
	return (&TagRankings{Query: &Query{
		Text: TAG_RANKINGS_BASE,
		Args: []interface{}{link_id, TAG_RANKINGS_PAGE_LIMIT},
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
LIMIT ?`

func (tr *TagRankings) Public() *TagRankings {
	tr.Text = strings.Replace(
		tr.Text,
		TAG_RANKINGS_BASE_FIELDS,
		TAG_RANKINGS_BASE_FIELDS+TAG_RANKINGS_PUBLIC_FIELDS,
		1,
	)

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
			Args: []interface{}{GLOBAL_CATS_PAGE_LIMIT},
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
SELECT global_cat, count(global_cat) as count
FROM GlobalCatsSplit
WHERE global_cat != ''
GROUP BY LOWER(global_cat)
ORDER BY count DESC, LOWER(global_cat) ASC
LIMIT ?`

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

	// With MATCH, reserved chars must be surrounded with ""
	match_arg :=
		// And singular/plural variants are added after escaping
		// reserved chars so that "(" and ")" are preserved
		WithOptionalPluralOrSingularForm(
			WithDoubleQuotesAroundReservedChars(
				cats[0],
			),
		)
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " +
			WithOptionalPluralOrSingularForm(
				WithDoubleQuotesAroundReservedChars(cats[i]),
			)
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

func (gcc *GlobalCatCounts) DuringPeriod(period string) *GlobalCatCounts {
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
			Args: []interface{}{
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
