package handler

import (
	"database/sql"
	"slices"
	"strings"

	"github.com/julianlk522/modeep/db"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"

	mutil "github.com/julianlk522/modeep/model/util"
)

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

func ScanTagRankings(tag_rankings_sql *query.TagRankings) (*[]model.TagRanking, error) {
	rows, err := db.Client.Query(tag_rankings_sql.Text, tag_rankings_sql.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tag_rankings := []model.TagRanking{}

	for rows.Next() {
		var tag model.TagRanking
		err = rows.Scan(
			&tag.LifeSpanOverlap,
			&tag.Cats,
		)
		if err != nil {
			return nil, err
		}
		tag_rankings = append(tag_rankings, tag)
	}

	return &tag_rankings, nil
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

// GetTopGlobalCats
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
	} else if a + "s" == b || b + "s" == a || a + "es" == b || b + "es" == a {
		return true
	} else if b + "s" == a || a + "s" == b || a + "es" == a || a + "es" == b {
		return true
	}

	return false
}

// AddTag
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

func TidyCats(cats string) string {
	split_cats := strings.Split(cats, ",")

	for i := 0; i < len(split_cats); i++ {
		split_cats[i] = strings.TrimSpace(split_cats[i])
	}

	slices.SortFunc(split_cats, func(i, j string) int {
		if strings.ToLower(i) < strings.ToLower(j) {
			return -1
		}
		return 1
	})

	return strings.Join(split_cats, ",")
}

// EditTag
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

// DeleteTag
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

	tags_for_link, err := ScanTagRankings(tag_rankings_sql)
	if err != nil {
		return err
	}

	cat_rankings := make(map[string]float32)
	var max_cat_score float32

	for _, tag := range *tags_for_link {
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

	if len(cat_rankings) > mutil.NUM_CATS_LIMIT {
		cat_rankings = LimitToTopCatRankings(cat_rankings)
	}

	var new_global_cats string
	for _, cat := range AlphabetizeCatRankings(cat_rankings) {
		if cat_rankings[cat] >= max_cat_score * (PERCENT_OF_MAX_CAT_SCORE_NEEDED_FOR_ASSIGNMENT / 100) {
			new_global_cats += cat + ","
		}
	}

	// remove trailing comma
	if len(new_global_cats) > 0 {
		new_global_cats = new_global_cats[:len(new_global_cats)-1]
	}

	if err = SetGlobalCats(link_id, new_global_cats); err != nil {
		return err
	}

	return nil
}

func LimitToTopCatRankings(cat_rankings map[string]float32) map[string]float32 {
	// should never happen but just in case...
	if len(cat_rankings) <= mutil.NUM_CATS_LIMIT {
		return cat_rankings
	}

	// sort (by value) before limit
	sorted_rankings := make([]model.CatRanking, 0, len(cat_rankings))
	for cat, score := range cat_rankings {
		sorted_rankings = append(sorted_rankings, model.CatRanking{
			Cat:   cat,
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

	limited_rankings := make(map[string]float32, mutil.NUM_CATS_LIMIT)
	for i := 0; i < mutil.NUM_CATS_LIMIT; i++ {
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

	tx, err := db.Client.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE Links 
		SET global_cats = ? 
		WHERE id = ?`,
		new_global_cats,
		link_id)
	if err != nil {
		return err
	}

	if err = IncrementSpellfixRanksForCats(tx, cats_diff.Added); err != nil {
		return err
	}
	if err = DecrementSpellfixRanksForCats(tx, cats_diff.Removed); err != nil {
		return err
	}

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
		Added:   added_cats,
		Removed: removed_cats,
	}, nil
}
