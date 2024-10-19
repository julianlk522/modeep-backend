package handler

import (
	"database/sql"
	"math"
	"slices"
	"strings"

	"github.com/julianlk522/fitm/db"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"

	"net/http"

	"github.com/go-chi/render"
)

// Get tags for link
func ScanTagPageLink[T model.Link | model.LinkSignedIn](link_sql *query.TagPageLink) (*T, error) {
	var link interface{}

	switch any(new(T)).(type) {
	case *model.Link:
		var l = &model.Link{}
		if err := db.Client.
			QueryRow(link_sql.Text, link_sql.Args...).
			Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.Summary,
				&l.SummaryCount,
				&l.LikeCount,
				&l.ImgURL,
			); err != nil {
			return nil, err
		}

		link = l
	case *model.LinkSignedIn:
		var l = &model.LinkSignedIn{}
		if err := db.Client.
			QueryRow(link_sql.Text, link_sql.Args...).
			Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.Summary,
				&l.SummaryCount,
				&l.LikeCount,
				&l.ImgURL,
				&l.IsLiked,
				&l.IsCopied,
			); err != nil {
			return nil, err
		}

		link = l
	}

	return link.(*T), nil
}

func GetUserTagForLink(login_name string, link_id string) (*model.Tag, error) {
	var id, cats, last_updated sql.NullString

	err := db.Client.
		QueryRow("SELECT id, cats, last_updated FROM 'Tags' WHERE submitted_by = ? AND link_id = ?;", login_name, link_id).
		Scan(&id, &cats, &last_updated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &model.Tag{
		ID:          id.String,
		Cats:        cats.String,
		LastUpdated: last_updated.String,
		LinkID:      link_id,
		SubmittedBy: login_name,
	}, nil
}

func ScanPublicTagRankings(tag_rankings_sql *query.TagRankings) (*[]model.TagRankingPublic, error) {
	rows, err := db.Client.Query(tag_rankings_sql.Text, tag_rankings_sql.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tag_rankings := []model.TagRankingPublic{}

	for rows.Next() {
		var tag model.TagRankingPublic
		err = rows.Scan(
			&tag.LifeSpanOverlap,
			&tag.Cats,
			&tag.SubmittedBy,
			&tag.LastUpdated,
		)
		if err != nil {
			return nil, err
		}
		tag_rankings = append(tag_rankings, tag)
	}

	return &tag_rankings, nil
}

// Get top global cats
func ScanGlobalCatCounts(global_cats_sql *query.GlobalCatCounts) (*[]model.CatCount, error) {
	if global_cats_sql.Error != nil {
		return nil, global_cats_sql.Error
	}

	rows, err := db.Client.Query(global_cats_sql.Text, global_cats_sql.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []model.CatCount

	for rows.Next() {
		var c model.CatCount
		err = rows.Scan(&c.Category, &c.Count)
		if err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}

	return &counts, nil
}

func RenderCatCounts(cat_counts *[]model.CatCount, w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.JSON(w, r, cat_counts)
}

// Add tag
func UserHasTaggedLink(login_name string, link_id string) (bool, error) {
	var t sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Tags WHERE submitted_by = ? AND link_id = ?;", login_name, link_id).Scan(&t)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil

}

func AlphabetizeCats(cats string) string {
	split_cats := strings.Split(cats, ",")
	slices.SortFunc(split_cats, func(i, j string) int {
		if strings.ToLower(i) < strings.ToLower(j) {
			return -1
		}
		return 1
	})

	return strings.Join(split_cats, ",")
}

// Edit tag
func UserSubmittedTagWithID(login_name string, tag_id string) (bool, error) {
	var t sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Tags WHERE submitted_by = ? AND id = ?;", login_name, tag_id).Scan(&t)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil

}

func GetLinkIDFromTagID(tag_id string) (string, error) {
	var link_id sql.NullString
	err := db.Client.QueryRow("SELECT link_id FROM Tags WHERE id = ?;", tag_id).Scan(&link_id)
	if err != nil {
		return "", err
	}

	return link_id.String, nil
}

// Delete tag
func TagExists(tag_id string) (bool, error) {
	var t sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Tags WHERE id = ?;", tag_id).Scan(&t)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func IsOnlyTag(tag_id string) (bool, error) {
	link_id, err := GetLinkIDFromTagID(tag_id)
	if err != nil {
		return false, err
	}

	rows, err := db.Client.Query("SELECT id FROM Tags WHERE link_id = ?;", link_id)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	rows.Next()
	if rows.Next() {
		return false, nil
	}

	return true, nil
}

// Calculate global cats
func CalculateAndSetGlobalCats(link_id string) error {
	overlap_scores_sql := query.NewTagRankings(link_id)
	if overlap_scores_sql.Error != nil {
		return overlap_scores_sql.Error
	}

	rows, err := db.Client.Query(overlap_scores_sql.Text, overlap_scores_sql.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	tag_rankings := []model.TagRanking{}
	for rows.Next() {
		var t model.TagRanking
		err = rows.Scan(&t.LifeSpanOverlap, &t.Cats)
		if err != nil {
			return err
		}
		tag_rankings = append(tag_rankings, t)
	}

	overlap_scores := make(map[string]float32)
	var max_cat_score float32

	for _, tag := range tag_rankings {

		// square root lifespan overlap to smooth out scores
		// (allows brand-new tags to still have some influence)
		tag.LifeSpanOverlap = float32(math.Sqrt(float64(tag.LifeSpanOverlap)))

		// multiple cats
		if strings.Contains(tag.Cats, ",") {
			cats := strings.Split(tag.Cats, ",")
			for _, cat := range cats {
				overlap_scores[cat] += tag.LifeSpanOverlap

				if overlap_scores[cat] > max_cat_score {
					max_cat_score = overlap_scores[cat]
				}
			}

			// single category
		} else {
			overlap_scores[tag.Cats] += tag.LifeSpanOverlap

			if overlap_scores[tag.Cats] > max_cat_score {
				max_cat_score = overlap_scores[tag.Cats]
			}
		}
	}

	// Alphabetize so global cats are assigned in order
	alphabetized_cats := AlphabetizeOverlapScoreCats(overlap_scores)

	// Assign to global cats if >= 25% of max category score
	var global_cats string
	for _, cat := range alphabetized_cats {
		if overlap_scores[cat] >= max_cat_score*0.25 {
			global_cats += cat + ","
		}
	}

	// Remove trailing comma
	if len(global_cats) > 0 && strings.HasSuffix(global_cats, ",") {
		global_cats = global_cats[:len(global_cats)-1]
	}

	err = SetGlobalCats(link_id, global_cats)
	if err != nil {
		return err
	}

	return nil
}

func AlphabetizeOverlapScoreCats(scores map[string]float32) []string {
	cats := make([]string, 0, len(scores))
	for cat := range scores {
		cats = append(cats, cat)
	}

	slices.SortFunc(cats, func(i, j string) int {
		if scores[i] > scores[j] {
			return -1
		} else if scores[i] == scores[j] && strings.ToLower(i) < strings.ToLower(j) {
			return -1
		}
		return 1
	})

	return cats
}

func SetGlobalCats(link_id string, text string) error {

	// determine diff to adjust spellfix ranks
	var old_cats_str string
	err := db.Client.QueryRow(
		"SELECT global_cats FROM Links WHERE id = ?;",
		link_id,
	).Scan(&old_cats_str)
	if err != nil {
		return err
	}

	var new_cats = strings.Split(text, ",")
	var old_cats = strings.Split(old_cats_str, ",")

	var added_cats []string
	for _, cat := range new_cats {
		if !slices.Contains(old_cats, cat) {
			added_cats = append(added_cats, cat)
		}
	}
	var removed_cats []string
	for _, cat := range old_cats {
		if !slices.Contains(new_cats, cat) {
			removed_cats = append(removed_cats, cat)
		}
	}

	// start transaction
	tx, err := db.Client.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// set link new cats
	_, err = tx.Exec(`
		UPDATE Links 
		SET global_cats = ? 
		WHERE id = ?`,
		text,
		link_id)
	if err != nil {
		return err
	}

	// update spellfix
	if err = IncrementSpellfixRanksForCats(tx, added_cats); err != nil {
		return err
	}
	if err = DecrementSpellfixRanksForCats(tx, removed_cats); err != nil {
		return err
	}

	// commit
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
