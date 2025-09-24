package query

import (
	"strings"

	mutil "github.com/julianlk522/modeep/model/util"
)

type Summaries struct {
	*Query
}

func NewSummariesForLink(link_id string) *Summaries {
	return (&Summaries{
		Query: &Query{
			Text: SUMMARIES_BASE_FIELDS +
				SUMMARIES_FROM +
				SUMMARIES_JOINS +
				SUMMARIES_GROUP_BY_AND_LIMIT,
			Args: []any{
				link_id,
				mutil.EARLIEST_STARRERS_LIMIT, 
				SUMMARIES_PAGE_LIMIT,
			},
		},
	})
}

const SUMMARIES_BASE_FIELDS = `SELECT 
	sumid, 
	text, 
	ln, 
	last_updated, 
	COALESCE(count(sl.id),0) as like_count,
	COALESCE(earliest_likers, "") as earliest_likers`

const SUMMARIES_FROM = ` 
FROM 
	(
	SELECT sumid, text, Users.login_name as ln, last_updated
	FROM 
		(
		SELECT id as sumid, text, submitted_by as sb, last_updated
		FROM Summaries
		WHERE link_id = ?
		) 
	JOIN Users 
	ON Users.id = sb
	)`

const SUMMARIES_JOINS = `
LEFT JOIN (
	SELECT 
		summary_id,
		GROUP_CONCAT(login_name, ', ') AS earliest_likers
	FROM (
		SELECT 
			sl.summary_id,
			u.login_name,
			ROW_NUMBER() OVER (PARTITION BY sl.summary_id ORDER BY sl.timestamp ASC) as row_num
		FROM "Summary Likes" sl
		JOIN Users u ON sl.user_id = u.id
		ORDER BY sl.timestamp ASC, u.login_name ASC
	)
	WHERE row_num <= ?
	GROUP BY summary_id
) EarliestLikers
ON EarliestLikers.summary_id = sumid
LEFT JOIN "Summary Likes" as sl 
ON sl.summary_id = sumid`

const SUMMARIES_GROUP_BY_AND_LIMIT = `
GROUP BY sumid
LIMIT ?;`

func (s *Summaries) AsSignedInUser(user_id string) *Summaries {
	s.Text = strings.Replace(
		s.Text,
		SUMMARIES_BASE_FIELDS,
		SUMMARIES_BASE_FIELDS+","+SUMMARIES_IS_LIKED_FIELD,
		1)

	s.Text = strings.Replace(
		s.Text,
		`LEFT JOIN "Summary Likes" as sl`,
		`LEFT JOIN
		(
		SELECT id, count(*) as is_liked, user_id, summary_id as slsumid
		FROM "Summary Likes"
		WHERE user_id = ?
		GROUP BY id
		)
	ON slsumid = sumid
	LEFT JOIN "Summary Likes" as sl`,
		1)

	// Pop limit arg
	s.Args = s.Args[0 : len(s.Args)-1]
	// Push user_id arg
	s.Args = append(s.Args, user_id)
	// Push limit arg back
	s.Args = append(s.Args, SUMMARIES_PAGE_LIMIT)

	return s
}

const SUMMARIES_IS_LIKED_FIELD = `
COALESCE(is_liked,0) as is_liked`
