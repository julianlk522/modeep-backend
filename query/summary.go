package query

import (
	"strings"

	mutil "github.com/julianlk522/fitm/model/util"
)

const SUMMARIES_PAGE_LIMIT = 20

type SummaryPageLink struct {
	*Query
}

func NewSummaryPageLink(ID string) *SummaryPageLink {
	return (&SummaryPageLink{
		Query: &Query{
			Text: SUMMARY_PAGE_LINK_BASE_FIELDS +
				SUMMARY_PAGE_LINK_BASE_FROM +
				SUMMARY_PAGE_LINK_BASE_JOINS,
			Args: []any{ID},
		},
	})
}

const SUMMARY_PAGE_LINK_BASE_FIELDS = `SELECT 
links_id as link_id, 
url, 
sb, 
sd, 
cats, 
summary, 
COALESCE(like_count,0) as like_count,
tag_count,   
img_file`

const SUMMARY_PAGE_LINK_BASE_FROM = ` 
FROM 
	(
	SELECT 
		id as links_id, 
		url, 
		submitted_by as sb, 
		submit_date as sd, 
		COALESCE(global_cats,"") as cats, 
		global_summary as summary, 
		COALESCE(img_file,"") as img_file 
	FROM Links
	WHERE id = ?
	)`

const SUMMARY_PAGE_LINK_BASE_JOINS = `
LEFT JOIN
	(
	SELECT count(*) as like_count, link_id as llink_id
	FROM "Link Likes"
	GROUP BY llink_id
	)
ON llink_id = links_id
LEFT JOIN 
	(
	SELECT count(*) as tag_count, link_id as tlink_id
	FROM Tags
	GROUP BY tlink_id
	)
ON tlink_id = links_id;`

func (spl *SummaryPageLink) AsSignedInUser(user_id string) *SummaryPageLink {
	spl.Text = strings.Replace(
		spl.Text,
		SUMMARY_PAGE_LINK_BASE_FIELDS,
		SUMMARY_PAGE_LINK_BASE_FIELDS+
			SUMMARY_PAGE_LINK_AUTH_FIELDS,
		1)

	spl.Text = strings.Replace(
		spl.Text, ";",
		SUMMARY_PAGE_LINK_AUTH_JOINS,
		1)
	spl.Args = append(spl.Args, user_id, user_id)

	return spl
}

const SUMMARY_PAGE_LINK_AUTH_FIELDS = `, 
	COALESCE(is_liked,0) as is_liked, 
	COALESCE(is_copied,0) as is_copied`

const SUMMARY_PAGE_LINK_AUTH_JOINS = ` 
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
				mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
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
