package query

import "github.com/julianlk522/modeep/db"

func NewTotals() *Query {
	return &Query{
		Text: `WITH LinksTotal AS (
			SELECT COUNT(*) AS link_count
			FROM Links
		),
		ClicksTotal AS (
			SELECT COUNT(*) AS click_count
			FROM Clicks
		),
		ContributorsTotal AS (
			SELECT COUNT(*) AS user_count
			FROM Users
			WHERE login_name != 'Auto Summary'
		),
		StarsTotal AS (
			SELECT COUNT(*) AS starred_count
			FROM Stars
		),
		TagsTotal AS (
			SELECT COUNT(*) AS tag_count
			FROM Tags
		),
		SummariesTotal AS (
			SELECT COUNT(*) AS summary_count
			FROM Summaries
			WHERE submitted_by != ?
		)
		SELECT *
		FROM LinksTotal
		CROSS JOIN ClicksTotal
		CROSS JOIN ContributorsTotal
		CROSS JOIN StarsTotal
		CROSS JOIN TagsTotal
		CROSS JOIN SummariesTotal;`,
		Args: []any{db.AUTO_SUMMARY_USER_ID},
	}
}
