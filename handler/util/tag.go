package handler

import (
	"database/sql"
	"slices"
	"strings"

	"github.com/julianlk522/modeep/db"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
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

func ScanTagRankings(tag_rankings_sql *query.TagRankingsForLink) (*[]model.TagRanking, error) {
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
func ScanGlobalCatCounts(global_cats_sql *query.TopGlobalCatCounts) (*[]model.CatCount, error) {
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

// Shared
func CatsResembleEachOther(a string, b string) bool {
	// Capitalization variants
	a, b = strings.ToLower(a), strings.ToLower(b)

	// Or singular/plural variants
	if a == b || a + "s" == b || b + "s" == a || a + "es" == b || b + "es" == a {
		return true
	} 

	return false
}

func TidyCats(cats string) string {
	split_cats := strings.Split(cats, ",")

	for i := range split_cats {
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

func CalculateAndSetGlobalCats(link_id string) error {
	global_cats_sql := query.NewGlobalCatsForLink(link_id)
	if global_cats_sql.Error != nil {
		return global_cats_sql.Error
	}
	
	var new_global_cats string
	err := db.Client.QueryRow(
		global_cats_sql.Text, 
		global_cats_sql.Args...).Scan(&new_global_cats)
	if err != nil {
		return err
	}

	if err = setGlobalCats(link_id, new_global_cats); err != nil {
		return err
	}

	return nil
}

func setGlobalCats(link_id string, new_global_cats string) error {
	cats_diff, err := getGlobalCatsDiff(link_id, new_global_cats)
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

func getGlobalCatsDiff(link_id string, new_cats_str string) (*model.GlobalCatsDiff, error) {
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
