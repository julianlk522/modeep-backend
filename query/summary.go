package query

import (
	"strings"
)

const SUMMARIES_PAGE_LIMIT = 20

// Summaries Page link
type SummaryPageLink struct {
	*Query
}

func NewSummaryPageLink(ID string) *SummaryPageLink {
	return (&SummaryPageLink{
		Query: &Query{
			Text: 
				SUMMARY_PAGE_LINK_BASE_FIELDS +
				SUMMARY_PAGE_LINK_BASE_FROM + 
				SUMMARY_PAGE_LINK_BASE_JOINS,
			Args: []interface{}{ID},
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
img_url`

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
		COALESCE(img_url,"") as img_url 
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

func (l *SummaryPageLink) AsSignedInUser(user_id string) *SummaryPageLink {
	l.Text = strings.Replace(
		l.Text, 
		SUMMARY_PAGE_LINK_BASE_FIELDS, 
		SUMMARY_PAGE_LINK_BASE_FIELDS + 
		SUMMARY_PAGE_LINK_AUTH_FIELDS, 
	1)

	l.Text = strings.Replace(
		l.Text, ";",
		SUMMARY_PAGE_LINK_AUTH_JOINS,
		1)
	l.Args = append(l.Args, user_id, user_id)

	return l
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

// Summaries for link
type Summaries struct {
	*Query
}

func NewSummariesForLink(link_id string) *Summaries {
	return (&Summaries{
		Query: &Query{
			Text: 
				SUMMARIES_BASE_FIELDS +
				SUMMARIES_FROM +
				SUMMARIES_JOIN +
				SUMMARIES_GBL,
			Args: []interface{}{link_id, SUMMARIES_PAGE_LIMIT},
		},
	})
}

const SUMMARIES_BASE_FIELDS = `SELECT 
	sumid, 
	text, 
	ln, 
	last_updated, 
	COALESCE(count(sl.id),0) as like_count`

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

const SUMMARIES_JOIN = `
LEFT JOIN "Summary Likes" as sl 
ON sl.summary_id = sumid`

const SUMMARIES_GBL = `
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

	// pop limit arg
	s.Args = s.Args[0 : len(s.Args)-1]
	// insert user_id arg
	s.Args = append(s.Args, user_id)
	// push limit arg back
	s.Args = append(s.Args, SUMMARIES_PAGE_LIMIT)

	return s
}

const SUMMARIES_IS_LIKED_FIELD = `
COALESCE(is_liked,0) as is_liked`
