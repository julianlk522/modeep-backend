package handler

import (
	"log"
	"net/http"

	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/handler/util"
	m "github.com/julianlk522/fitm/middleware"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

func GetLinks(w http.ResponseWriter, r *http.Request) {
	links_sql := query.NewTopLinks()

	// cats
	cats_params := r.URL.Query().Get("cats")
	if cats_params != "" {
		cats := strings.Split(cats_params, ",")
		links_sql = links_sql.FromCats(cats)
	}

	// period
	period_params := r.URL.Query().Get("period")
	if period_params != "" {
		links_sql = links_sql.DuringPeriod(period_params)
	}

	// sort by
	sort_params := r.URL.Query().Get("sort_by")
	if sort_params != "" {
		links_sql = links_sql.SortBy(sort_params)
	}

	// auth fields
	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["user_id"].(string)
	if req_user_id != "" {
		links_sql = links_sql.AsSignedInUser(req_user_id)
	}

	// nsfw
	var nsfw_params string
	if r.URL.Query().Get("nsfw") != "" {
		nsfw_params = r.URL.Query().Get("nsfw")
	} else if r.URL.Query().Get("NSFW") != "" {
		nsfw_params = r.URL.Query().Get("NSFW")
	}

	if nsfw_params == "true" {
		links_sql = links_sql.NSFW()
	} else if nsfw_params != "false" && nsfw_params != "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrInvalidNSFWParams))
		return
	}

	// pagination
	page := r.Context().Value(m.PageKey).(int)
	links_sql = links_sql.Page(page)

	if links_sql.Error != nil {
		render.Render(w, r, e.ErrInvalidRequest(links_sql.Error))
		return
	}

	// scan
	if req_user_id != "" {
		links, err := util.ScanLinks[model.LinkSignedIn](links_sql)
		if err != nil {
			render.Render(w, r, e.Err500(err))
		}
		render.JSON(w, r, util.PaginateLinks(links, page))
	} else {
		links, err := util.ScanLinks[model.Link](links_sql)
		if err != nil {
			render.Render(w, r, e.Err500(err))
		}
		render.JSON(w, r, util.PaginateLinks(links, page))
	}
}

func AddLink(w http.ResponseWriter, r *http.Request) {
	request := &model.NewLinkRequest{}
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["login_name"].(string)
	
	// verify user has not already submitted too many links today
	if user_submitted_max_daily_links, err := util.UserHasSubmittedMaxDailyLinks(req_login_name); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	} else if user_submitted_max_daily_links {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrMaxDailyLinkSubmissionsReached(util.MAX_DAILY_LINKS)))
		return
	}

	if util.IsYouTubeVideoLink(request.NewLink.URL) {
		if err := util.ObtainYouTubeMetaData(request); err != nil {

			// if unable to get YT metadata, try treating as normal URL
			// (in case of, e.g., example.com/youtube.com/watch?v=1234
			// though this should not happen per util.TestIsYouTubeVideoLink cases)
			if err = util.ObtainURLMetaData(request); err != nil {
				render.Render(w, r, e.ErrInvalidRequest(err))
				return
			}
		}

	} else {
		if err := util.ObtainURLMetaData(request); err != nil {
			render.Render(w, r, e.ErrInvalidRequest(err))
			return
		}
	}

	// verify URL is unique
	// this comes after ResolveURL() because may mutate slightly
	if is_duplicate, dupe_link_id := util.LinkAlreadyAdded(request.URL); is_duplicate {
		render.Status(r, http.StatusConflict)
		render.Render(w, r, e.ErrInvalidRequest(e.ErrDuplicateLink(request.URL, dupe_link_id)))
		return
	}

	// Verified: add link
	request.SubmittedBy = req_login_name

	// sort cats
	request.Cats = util.AlphabetizeCats(request.NewLink.Cats)

	// Start Transaction
	tx, err := db.Client.Begin()
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}
	defer tx.Rollback()

	// insert summary(ies)
	// (might have user-submitted, auto, or both)
	// auto summary
	if request.AutoSummary != "" {
		_, err := tx.Exec(
			"INSERT INTO Summaries VALUES(?,?,?,?,?);",
			uuid.New().String(),
			request.AutoSummary,
			request.ID,
			db.AUTO_SUMMARY_USER_ID,
			request.SubmitDate,
		)
		if err != nil {
			// continue... no auto summary
			// but log err
			log.Print("Error adding auto summary: ", err)
		} else {
			request.SummaryCount = 1
		}
	}

	// user summary
	if request.NewLink.Summary != "" {
		req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["user_id"].(string)
		_, err := tx.Exec(
			"INSERT INTO Summaries VALUES(?,?,?,?,?);",
			uuid.New().String(),
			request.NewLink.Summary,
			request.ID,
			req_user_id,
			request.SubmitDate,
		)
		if err != nil {
			render.Render(w, r, e.Err500(err))
			return
		} else {
			request.SummaryCount += 1
		}
	}

	// insert tag
	_, err = tx.Exec(
		"INSERT INTO Tags VALUES(?,?,?,?,?);",
		uuid.New().String(),
		request.ID,
		request.Cats,
		request.SubmittedBy,
		request.SubmitDate,
	)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	if request.NewLink.Summary != "" {
		request.Summary = request.NewLink.Summary
	} else if request.AutoSummary != "" {
		request.Summary = request.AutoSummary
	} else {
		request.Summary = ""
	}

	// insert link
	_, err = tx.Exec(
		"INSERT INTO Links VALUES(?,?,?,?,?,?,?);",
		request.ID,
		request.URL,
		request.SubmittedBy,
		request.SubmitDate,
		request.Cats,
		request.Summary,
		request.ImgURL,
	)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	// increment spellfix ranks
	err = util.IncrementSpellfixRanksForCats(
		tx,
		strings.Split(request.Cats, ","),
	)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	// Commit
	if err = tx.Commit(); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	// Return new link
	new_link := model.Link{
		ID:           request.ID,
		URL:          request.URL,
		SubmittedBy:  request.SubmittedBy,
		SubmitDate:   request.SubmitDate,
		Cats:         request.Cats,
		Summary:      request.Summary,
		SummaryCount: request.SummaryCount,
		ImgURL:       request.ImgURL,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, new_link)
}

func DeleteLink(w http.ResponseWriter, r *http.Request) {
	request := &model.DeleteLinkRequest{}
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	link_exists, err := util.LinkExists(request.LinkID)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	} else if !link_exists {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLinkWithID))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["login_name"].(string)
	if !util.UserSubmittedLink(req_login_name, request.LinkID) {
		render.Render(w, r, e.ErrUnauthorized(e.ErrDoesntOwnLink))
		return
	}

	// fetch global cats before deleting
	// (to properly update spellfix ranks)
	var gc string
	err = db.Client.QueryRow("SELECT global_cats FROM Links WHERE id = ?;", request.LinkID).Scan(&gc)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	// start transaction
	tx, err := db.Client.Begin()
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}
	defer tx.Rollback()

	// delete
	_, err = tx.Exec(
		"DELETE FROM Links WHERE id = ?;",
		request.LinkID,
	)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	// update spellfix
	err = util.DecrementSpellfixRanksForCats(
		tx,
		strings.Split(gc, ","),
	)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	// commit
	if err = tx.Commit(); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	w.WriteHeader(http.StatusResetContent)
}

func LikeLink(w http.ResponseWriter, r *http.Request) {
	link_id := chi.URLParam(r, "link_id")
	if link_id == "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLinkID))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["login_name"].(string)
	if util.UserSubmittedLink(req_login_name, link_id) {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrCannotLikeOwnLink))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["user_id"].(string)
	if util.UserHasLikedLink(req_user_id, link_id) {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrLinkAlreadyLiked))
		return
	}

	new_like_id := uuid.New().String()
	_, err := db.Client.Exec(
		`INSERT INTO "Link Likes" VALUES(?,?,?);`,
		new_like_id,
		link_id,
		req_user_id,
	)
	if err != nil {
		render.Render(w, r, e.Err500(err))
	}

	w.WriteHeader(http.StatusNoContent)
}

func UnlikeLink(w http.ResponseWriter, r *http.Request) {
	link_id := chi.URLParam(r, "link_id")
	if link_id == "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLinkID))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["user_id"].(string)
	if !util.UserHasLikedLink(req_user_id, link_id) {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrLinkNotLiked))
		return
	}

	_, err := db.Client.Exec(
		`DELETE FROM "Link Likes" WHERE link_id = ? AND user_id = ?;`,
		link_id,
		req_user_id,
	)
	if err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func CopyLink(w http.ResponseWriter, r *http.Request) {
	link_id := chi.URLParam(r, "link_id")
	if link_id == "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLinkID))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["login_name"].(string)
	owns_link := util.UserSubmittedLink(req_login_name, link_id)
	if owns_link {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrCannotCopyOwnLink))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["user_id"].(string)
	already_copied := util.UserHasCopiedLink(req_user_id, link_id)
	if already_copied {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrLinkAlreadyCopied))
		return
	}

	new_copy_id := uuid.New().String()

	_, err := db.Client.Exec(
		`INSERT INTO "Link Copies" VALUES(?,?,?);`,
		new_copy_id,
		link_id,
		req_user_id,
	)
	if err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func UncopyLink(w http.ResponseWriter, r *http.Request) {
	link_id := chi.URLParam(r, "link_id")
	if link_id == "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLinkID))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]interface{})["user_id"].(string)
	already_copied := util.UserHasCopiedLink(req_user_id, link_id)
	if !already_copied {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrLinkNotCopied))
		return
	}

	// Delete
	_, err := db.Client.Exec(
		`DELETE FROM "Link Copies" WHERE link_id = ? AND user_id = ?;`,
		link_id,
		req_user_id,
	)
	if err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusNoContent)
}
