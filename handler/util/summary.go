package handler

import (
	"database/sql"
	"log"

	"net/http"

	"github.com/julianlk522/modeep/db"
	e "github.com/julianlk522/modeep/error"
	m "github.com/julianlk522/modeep/middleware"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func BuildSummaryPageForLink(link_id string, r *http.Request) (any, error) {
	get_link_sql := query.NewSingleLink(link_id)
	get_summaries_sql := query.NewSummariesForLink(link_id)

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	if req_user_id != "" {
		get_link_sql = get_link_sql.AsSignedInUser(req_user_id)
		get_summaries_sql = get_summaries_sql.AsSignedInUser(req_user_id)
	}

	if get_link_sql.Error != nil {
		return nil, get_link_sql.Error
	} else if get_summaries_sql.Error != nil {
		return nil, get_summaries_sql.Error
	}

	if req_user_id != "" {
		l, err := ScanSingleLink[model.LinkSignedIn](get_link_sql)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, e.ErrNoLinkWithID
			} else {
				return nil, err
			}
		}

		rows, err := get_summaries_sql.ValidateAndExecuteRows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		summaries := []model.SummarySignedIn{}
		for rows.Next() {
			s := model.SummarySignedIn{}
			err := rows.Scan(
				&s.ID,
				&s.Text,
				&s.SubmittedBy,
				&s.LastUpdated,
				&s.LikeCount,
				&s.EarliestLikers,
				&s.IsLiked,
			)
			if err != nil {
				return nil, err
			}
			summaries = append(summaries, s)
		}

		l.SummaryCount = len(summaries)

		return model.SummaryPage[model.SummarySignedIn, model.LinkSignedIn]{
			Link:      *l,
			Summaries: summaries,
		}, nil

	} else {
		l, err := ScanSingleLink[model.Link](get_link_sql)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, e.ErrNoLinkWithID
			} else {
				return nil, err
			}
		}

		rows, err := get_summaries_sql.ValidateAndExecuteRows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		summaries := []model.Summary{}
		for rows.Next() {
			s := model.Summary{}
			err := rows.Scan(
				&s.ID,
				&s.Text,
				&s.SubmittedBy,
				&s.LastUpdated,
				&s.LikeCount,
				&s.EarliestLikers,
			)
			if err != nil {
				return nil, err
			}
			summaries = append(summaries, s)
		}

		l.SummaryCount = len(summaries)

		return model.SummaryPage[model.Summary, model.Link]{
			Link:      *l,
			Summaries: summaries,
		}, nil
	}
}

// Add summary
func LinkExists(link_id string) (bool, error) {
	var l sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Links WHERE id = ?", link_id).Scan(&l)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return l.Valid, nil
}

func GetIDOfUserSummaryForLink(user_id string, link_id string) (string, error) {
	var summary_id sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Summaries WHERE submitted_by = ? AND link_id = ?", user_id, link_id).Scan(&summary_id)

	if err != nil {
		return "", err
	}

	return summary_id.String, nil
}

// Delete summary
func GetLinkIDFromSummaryID(summary_id string) (string, error) {
	var lid sql.NullString
	err := db.Client.QueryRow(`SELECT link_id FROM Summaries WHERE id = ?`, summary_id).Scan(&lid)
	if err != nil {
		return "", err
	}

	return lid.String, nil
}

func LinkHasOneSummaryLeft(link_id string) (bool, error) {
	var c sql.NullInt32
	err := db.Client.QueryRow("SELECT COUNT(id) FROM Summaries WHERE link_id = ?", link_id).Scan(&c)
	if err != nil {
		return false, err
	}

	return c.Int32 == 1, nil
}

// Like / unlike summary
func SummarySubmittedByUser(summary_id string, user_id string) (bool, error) {
	var submitted_by sql.NullString
	err := db.Client.QueryRow("SELECT submitted_by FROM Summaries WHERE id = ?", summary_id).Scan(&submitted_by)

	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return submitted_by.String == user_id, nil
}

func IsAutoSummaryForLinkSubmittedByUser(summary_id string, user_id string) (bool, error) {
	is_auto_summary_sql := `SELECT 1
FROM Summaries
WHERE id = ?
AND submitted_by = ?
AND link_id IN (
	SELECT id
	FROM Links
	WHERE submitted_by IN (
		SELECT login_name
		FROM Users
		WHERE id = ?
	)
);`
	var is_auto_summary_for_their_link sql.NullInt16
	if err := db.Client.QueryRow(
		is_auto_summary_sql,
		summary_id,
		db.AUTO_SUMMARY_USER_ID,
		user_id,
	).Scan(&is_auto_summary_for_their_link); err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return is_auto_summary_for_their_link.Int16 == 1, nil
}

func UserHasLikedSummary(user_id string, summary_id string) (bool, error) {
	var summary_like_id sql.NullString

	err := db.Client.QueryRow(`SELECT id 
		FROM "Summary Likes" 
		WHERE user_id = ? 
		AND summary_id = ?`,
		user_id,
		summary_id,
	).Scan(&summary_like_id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return summary_like_id.Valid, nil
}

func CalculateAndSetGlobalSummary(link_id string) error {
	// If there are no summaries then should just be empty string
	var summaries_count_for_link sql.NullInt32
	err := db.Client.QueryRow("SELECT COUNT(id) FROM Summaries WHERE link_id = ?", link_id).Scan(&summaries_count_for_link)
	if err != nil {
		return err
	}

	if summaries_count_for_link.Int32 == 0 {
		_, err = db.Client.Exec(`UPDATE Links SET global_summary = '' WHERE id = ?`, link_id)
		return err
	}

	// Otherwise global summary is set to summary with most upvotes
	// unless 1st is auto summary and is tied with 2nd place, then use 2nd place
	var top_summary_text string
	err = db.Client.QueryRow(`WITH RankedSummaries AS (
		SELECT 
			s.text,
			s.submitted_by,
			COALESCE(sl.like_count, 0) AS like_count,
			ROW_NUMBER() OVER (
			ORDER BY 
				COALESCE(sl.like_count, 0) DESC, 
				CASE WHEN s.submitted_by = ? THEN 1 ELSE 0 END,
				s.text ASC
			) AS rank
		FROM Summaries s
		LEFT JOIN (
			SELECT summary_id, COUNT(*) AS like_count
			FROM "Summary Likes"
			GROUP BY summary_id
		) sl ON s.id = sl.summary_id
		WHERE s.link_id = ?
	)
	SELECT text
	FROM RankedSummaries
	WHERE rank = 1`,
		db.AUTO_SUMMARY_USER_ID,
		link_id,
	).Scan(&top_summary_text)

	if err != nil {
		return err
	}

	// Set global summary if not already set to top result
	var gs string
	err = db.Client.QueryRow(`SELECT global_summary 
		FROM Links 
		WHERE id = ?`,
		link_id).Scan(&gs)
	if err != nil {
		return err
	} else if gs == "" || gs != top_summary_text {
		setLinkGlobalSummary(link_id, top_summary_text)
	}

	return nil
}

func setLinkGlobalSummary(link_id string, text string) {
	_, err := db.Client.Exec(`UPDATE Links SET global_summary = ? WHERE id = ?`, text, link_id)
	if err != nil {
		log.Fatal(err)
	}
}
