package query

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
		),
		LikesTotal AS (
			SELECT COUNT(*) AS like_count
			FROM "Link Likes"
		),
		TagsTotal AS (
			SELECT COUNT(*) AS tag_count
			FROM Tags
		),
		SummariesTotal AS (
			SELECT COUNT(*) AS summary_count
			FROM Summaries
		)
		SELECT *
		FROM LinksTotal
		CROSS JOIN ClicksTotal
		CROSS JOIN ContributorsTotal
		CROSS JOIN LikesTotal
		CROSS JOIN TagsTotal
		CROSS JOIN SummariesTotal;`,
	}
}