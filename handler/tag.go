package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/julianlk522/modeep/db"
	e "github.com/julianlk522/modeep/error"
	util "github.com/julianlk522/modeep/handler/util"
	m "github.com/julianlk522/modeep/middleware"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func GetTagPage(w http.ResponseWriter, r *http.Request) {
	link_id := chi.URLParam(r, "link_id")
	if link_id == "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLinkID))
		return
	}

	link_exists, err := util.LinkExists(link_id)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	} else if !link_exists {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLinkWithID))
		return
	}

	// refresh global cats before querying
	util.CalculateAndSetGlobalCats(link_id)

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	link_sql := query.NewSingleLink(link_id)
	if req_user_id != "" {
		link_sql = link_sql.AsSignedInUser(req_user_id)
	}
	if link_sql.Error != nil {
		render.Render(w, r, e.ErrInvalidRequest(link_sql.Error))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]any)["login_name"].(string)
	user_tag, err := util.GetUserTagForLink(req_login_name, link_id)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	tag_rankings_sql := query.NewTagRankingsForLink(link_id)
	if tag_rankings_sql.Error != nil {
		render.Render(w, r, e.ErrInvalidRequest(tag_rankings_sql.Error))
		return
	}

	tag_rankings, err := util.ScanTagRankings(tag_rankings_sql)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	if req_user_id != "" {
		link, err := util.ScanSingleLink[model.LinkSignedIn](link_sql)
		if err != nil {
			render.Render(w, r, e.ErrInvalidRequest(err))
			return
		}

		render.JSON(w, r, model.TagPage[model.LinkSignedIn]{
			Link:        link,
			UserTag:     user_tag,
			TagRankings: tag_rankings,
		})

	} else {
		link, err := util.ScanSingleLink[model.Link](link_sql)
		if err != nil {
			render.Render(w, r, e.ErrInvalidRequest(err))
			return
		}

		render.JSON(w, r, model.TagPage[model.Link]{
			Link:        link,
			UserTag:     user_tag,
			TagRankings: tag_rankings,
		})
	}

}

func GetTopGlobalCats(w http.ResponseWriter, r *http.Request) {
	opts, err := util.GetTopGlobalCatsOptionsFromRequestParams(r.URL.Query())
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}
	global_cats_sql, err := query.NewTopGlobalCatCounts().FromOptions(opts)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	counts, err := util.ScanGlobalCatCounts(global_cats_sql)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	// If "more" params passed: check cat counts for possible merged
	// plural/singular spelling variations.

	// On other pages with links we can search link-by-link for merged cats,
	// which is more reliable, but on the /more page there are only cat counts.

	// (It's more accurate to check links because they are bundled with their
	// tag; you can verify whether the cats in search results were included
	// only because 1+ from the same tag was a close approximation for a cat
	// filter (AKA merged) OR if it's tag also included a cat filter exactly
	// and required no merged spelling variants to cause it to be included.

	// So this approach isn't 100% reliable but meh close enough for now.)
	query_params := r.URL.Query()
	more_params := query_params.Get("more")
	cats_params := query_params.Get("cats")

	if more_params == "true" && cats_params != "" {
		cat_filters := strings.Split(cats_params, ",")
		merged_cats := []string{}

		for _, count := range *counts {
			for _, cf := range cat_filters {
				if util.CatsResembleEachOther(count.Category, cf) {
					merged_cats = append(merged_cats, count.Category)
				}
			}
		}

		counts_and_merges := struct {
			Counts     []model.CatCount
			MergedCats []string
		}{
			Counts:     *counts,
			MergedCats: merged_cats,
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, counts_and_merges)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, counts)
}

func GetSpellfixMatchesForSnippet(w http.ResponseWriter, r *http.Request) {
	snippet := chi.URLParam(r, "*")
	if snippet == "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoCatsSnippet))
		return
	}
	opts, err := util.GetSpellfixOptionsFromRequestParams(r.URL.Query())
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}
	spfx_sql, err := query.NewSpellfixMatchesForSnippet(snippet).FromOptions(opts)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}
	rows, err := spfx_sql.ValidateAndExecuteRows()
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}
	defer rows.Close()

	var matches []model.CatCount
	for rows.Next() {
		var word string
		var rank int32
		if err := rows.Scan(&word, &rank); err != nil {
			render.Render(w, r, e.ErrInternalServerError(err))
			return
		}
		matches = append(matches, model.CatCount{
			Category: word,
			Count:    rank,
		})
	}

	render.JSON(w, r, matches)
	render.Status(r, http.StatusOK)
}

func AddTag(w http.ResponseWriter, r *http.Request) {
	tag_data := &model.NewTagRequest{}
	if err := render.Bind(r, tag_data); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	link_exists, err := util.LinkExists(tag_data.LinkID)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	} else if !link_exists {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLinkWithID))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]any)["login_name"].(string)
	duplicate, err := util.UserHasTaggedLink(req_login_name, tag_data.LinkID)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	} else if duplicate {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrDuplicateTag))
		return
	}

	tag_data.Cats = util.TidyCats(tag_data.Cats)

	_, err = db.Client.Exec(
		"INSERT INTO Tags VALUES(?,?,?,?,?);",
		tag_data.ID,
		tag_data.LinkID,
		tag_data.Cats,
		req_login_name,
		tag_data.LastUpdated,
	)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	if err = util.CalculateAndSetGlobalCats(tag_data.LinkID); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, tag_data)
}

func EditTag(w http.ResponseWriter, r *http.Request) {
	edit_tag_data := &model.EditTagRequest{}
	if err := render.Bind(r, edit_tag_data); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]any)["login_name"].(string)
	owns_tag, err := util.UserSubmittedTagWithID(req_login_name, edit_tag_data.ID)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoTagWithID))
		return
	} else if !owns_tag {
		render.Render(w, r, e.ErrForbidden(e.ErrDoesntOwnTag))
		return
	}

	edit_tag_data.Cats = util.TidyCats(edit_tag_data.Cats)

	if _, err = db.Client.Exec(
		`UPDATE Tags 
		SET cats = ?, 
		last_updated = ? 
		WHERE id = ?;`,
		edit_tag_data.Cats,
		edit_tag_data.LastUpdated,
		edit_tag_data.ID,
	); err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	link_id, err := util.GetLinkIDFromTagID(edit_tag_data.ID)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	} else if err = util.CalculateAndSetGlobalCats(link_id); err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, edit_tag_data)
}

func DeleteTag(w http.ResponseWriter, r *http.Request) {
	delete_tag_data := &model.DeleteTagRequest{}
	if err := render.Bind(r, delete_tag_data); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	tag_exists, err := util.TagExists(delete_tag_data.ID)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	} else if !tag_exists {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoTagWithID))
		return
	}

	is_only_tag, err := util.IsOnlyTag(delete_tag_data.ID)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	} else if is_only_tag {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrCantDeleteOnlyTag))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]any)["login_name"].(string)
	owns_tag, err := util.UserSubmittedTagWithID(req_login_name, delete_tag_data.ID)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	} else if !owns_tag {
		render.Render(w, r, e.ErrForbidden(e.ErrDoesntOwnTag))
		return
	}

	// must get link_id before deleting
	link_id, err := util.GetLinkIDFromTagID(delete_tag_data.ID)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	_, err = db.Client.Exec(
		"DELETE FROM Tags WHERE id = ?;",
		delete_tag_data.ID,
	)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	if err = util.CalculateAndSetGlobalCats(link_id); err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
