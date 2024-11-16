package handler

import (
	"database/sql"
	"slices"
	"strings"

	"github.com/julianlk522/fitm/db"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

const MAX_TAG_CATS = 15

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

func CatsAreSingularOrPluralVariationsOfEachOther(a string, b string) bool {
	if a == b {
		return false
	}

	if a + "s" == b || b + "s" == a || a+"es" == b || b+"es" == a {
		return true
	}

	return false
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

func CalculateAndSetGlobalCats(link_id string) error {
	tag_rankings_sql := query.NewTagRankings(link_id)
	if tag_rankings_sql.Error != nil {
		return tag_rankings_sql.Error
	}

	rows, err := db.Client.Query(tag_rankings_sql.Text, tag_rankings_sql.Args...)
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

	cat_rankings := make(map[string]float32)
	var max_cat_score float32

	for _, tag := range tag_rankings {

		// multiple cats
		if strings.Contains(tag.Cats, ",") {
			cats := strings.Split(tag.Cats, ",")
			for _, cat := range cats {
				cat_rankings[cat] += tag.LifeSpanOverlap

				if cat_rankings[cat] > max_cat_score {
					max_cat_score = cat_rankings[cat]
				}
			}

		// single cat
		} else {
			cat_rankings[tag.Cats] += tag.LifeSpanOverlap

			if cat_rankings[tag.Cats] > max_cat_score {
				max_cat_score = cat_rankings[tag.Cats]
			}
		}
	}

	// limit if more than MAX_TAG_CATS cats
	if len(cat_rankings) > MAX_TAG_CATS {
		cat_rankings = LimitToTopCatRankings(cat_rankings)
	}

	// Alphabetize so global cats are assigned in order
	alphabetized_cats := AlphabetizeCatRankings(cat_rankings)

	// Assign to global cats if >= 25% of max category score
	var global_cats string
	for _, cat := range alphabetized_cats {
		if cat_rankings[cat] >= max_cat_score*0.25 {
			global_cats += cat + ","
		}
	}
	if len(global_cats) > 0 {
		global_cats = global_cats[:len(global_cats)-1]
	}

	err = SetGlobalCats(link_id, global_cats)
	if err != nil {
		return err
	}

	return nil
}

func LimitToTopCatRankings(cat_rankings map[string]float32) map[string]float32 {

	// should never happen but just in case...
	if len(cat_rankings) <= MAX_TAG_CATS {
		return cat_rankings
	}

	// sort by values before limit
	sorted_rankings := make([]model.CatRanking, 0, len(cat_rankings))
	for cat, score := range cat_rankings {
		sorted_rankings = append(sorted_rankings, model.CatRanking{
			Cat:  cat,
			Score: score,
		})
	}
	slices.SortFunc(sorted_rankings, func(i, j model.CatRanking) int {
		if i.Score > j.Score {
			return -1
		} else if i.Score == j.Score && strings.ToLower(i.Cat) < strings.ToLower(j.Cat) {
			return -1
		}
		return 1
	})

	limited_rankings := make(map[string]float32, MAX_TAG_CATS)
	for i := 0; i < MAX_TAG_CATS; i++ {
		limited_rankings[sorted_rankings[i].Cat] = sorted_rankings[i].Score
	}
	
	return limited_rankings
}

func AlphabetizeCatRankings(scores map[string]float32) []string {
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

func SetGlobalCats(link_id string, new_global_cats string) error {
	cats_diff, err := GetGlobalCatsDiff(link_id, new_global_cats)
	if err != nil {
		return err
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
		new_global_cats,
		link_id)
	if err != nil {
		return err
	}

	// update spellfix
	if err = IncrementSpellfixRanksForCats(tx, cats_diff.Added); err != nil {
		return err
	}
	if err = DecrementSpellfixRanksForCats(tx, cats_diff.Removed); err != nil {
		return err
	}

	// commit
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func GetGlobalCatsDiff(link_id string, new_cats_str string) (*model.GlobalCatsDiff, error) {
	var old_cats_str string
	err := db.Client.QueryRow(
		"SELECT global_cats FROM Links WHERE id = ?;",
		link_id,
	).Scan(&old_cats_str)
	if err != nil {
		return nil, err
	}

	var new_cats = strings.Split(new_cats_str, ",")
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

	return &model.GlobalCatsDiff{
		Added: added_cats,
		Removed: removed_cats,
	}, nil
}