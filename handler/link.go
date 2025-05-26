package handler

import (
	"log"
	"net/http"
	"os"

	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/handler/util"
	m "github.com/julianlk522/fitm/middleware"
	"github.com/julianlk522/fitm/model"
	mutil "github.com/julianlk522/fitm/model/util"
	"github.com/julianlk522/fitm/query"
)

func GetLinks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	links_sql := query.
		NewTopLinks().
		FromRequestParams(
			r.URL.Query(),
		)

	req_user_id := ctx.Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	if req_user_id != "" {
		links_sql = links_sql.AsSignedInUser(req_user_id)
	}

	page := ctx.Value(m.PageKey).(int)
	links_sql = links_sql.Page(page)

	if links_sql.Error != nil {
		render.Render(w, r, e.ErrInvalidRequest(links_sql.Error))
		return
	}

	var resp any
	var err error
	cats_params := r.URL.Query().Get("cats")

	if req_user_id != "" {
		resp, err = util.PrepareLinksResponse[model.LinkSignedIn](links_sql, cats_params)
	} else {
		resp, err = util.PrepareLinksResponse[model.Link](links_sql, cats_params)
	}
	if err != nil {
		render.Render(w, r, e.Err500(err))
	}

	render.JSON(w, r, resp)
}

func GetPreviewImg(w http.ResponseWriter, r *http.Request) {
	var file_name string = chi.URLParam(r, "file_name")
	path := util.Preview_img_dir + "/" + file_name

	if _, err := os.Stat(path); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrPreviewImgNotFound))
		return
	}

	http.ServeFile(w, r, path)
}

func AddLink(w http.ResponseWriter, r *http.Request) {
	request := &model.NewLinkRequest{}
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]any)["login_name"].(string)

	if user_submitted_max_daily_links, err := util.UserHasSubmittedMaxDailyLinks(req_login_name); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	} else if user_submitted_max_daily_links {
		render.Render(w, r, e.ErrTooManyRequests(e.ErrMaxDailyLinkSubmissionsReached(util.MAX_DAILY_LINKS)))
		return
	}

	// test URL response
	var resp *http.Response
	resp, err := util.GetResolvedURLResponse(request.URL)
	if err != nil {
		render.Render(w, r, e.ErrUnprocessable(err))
		return
	}
	log.Printf("resp.StatusCode: %d", resp.StatusCode)
	defer resp.Body.Close()

	// save adjusted URL (after any redirects e.g., to wwww.)
	// unless modified due to 302/401/403/429 etc. redirect
	url_after_redirects := resp.Request.URL.String()
	var final_url string

	is_302_redirect := resp.StatusCode == http.StatusFound
	is_unauthorized := resp.StatusCode == http.StatusUnauthorized
	is_forbidden := resp.StatusCode == http.StatusForbidden
	is_too_many_requests := resp.StatusCode == http.StatusTooManyRequests
	is_google_sorry_page := strings.Contains(url_after_redirects, "google.com/sorry")

	if !(is_302_redirect || is_unauthorized || is_forbidden || is_too_many_requests || is_google_sorry_page) {
		final_url = strings.TrimSuffix(url_after_redirects, "/")
	} else {
		final_url = strings.TrimSuffix(request.URL, "/")
	}

	if is_duplicate, link_id := util.LinkAlreadyAdded(final_url); is_duplicate {
		render.Status(r, http.StatusConflict)
		render.Render(w, r, e.ErrConflict(e.ErrDuplicateLink(final_url, link_id)))
		return
	}

	var new_link = &model.NewLink{
		SubmittedBy:    req_login_name,
		NewLinkRequest: &model.NewLinkRequest{},
	}
	var x_md *model.LinkExtraMetadata

	if util.IsYTVideo(final_url) {
		if yt_md, err := util.GetYTVideoMetadata(final_url); err == nil {
			new_link.URL = "https://www.youtube.com/watch?v=" + yt_md.ID
			new_link.AutoSummary = yt_md.Items[0].Snippet.Title
			new_link.PreviewImgURL = yt_md.Items[0].Snippet.Thumbnails.Default.URL
		} else {
			x_md = util.GetLinkExtraMetadataFromResponse(resp)
		}
	} else {
		x_md = util.GetLinkExtraMetadataFromResponse(resp)
	}
	if x_md != nil {
		new_link.AutoSummary = x_md.AutoSummary
		new_link.PreviewImgURL = x_md.PreviewImgURL
	}

	// Verified: add link
	tx, err := db.Client.Begin()
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}
	defer tx.Rollback()

	new_link.LinkID = request.LinkID
	new_link.SubmitDate = request.SubmitDate

	// Insert auto summary
	if new_link.AutoSummary != "" {
		if _, err := tx.Exec(
			"INSERT INTO Summaries VALUES(?,?,?,?,?);",
			uuid.New().String(),
			new_link.AutoSummary,
			new_link.LinkID,
			db.AUTO_SUMMARY_USER_ID,
			new_link.SubmitDate,
		); err != nil {
			log.Print("Error adding auto summary: ", err)
		} else {
			new_link.SummaryCount = 1
		}
	}

	// Insert summary
	new_link.Summary = request.Summary
	if new_link.Summary != "" {
		req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
		if _, err := tx.Exec(
			"INSERT INTO Summaries VALUES(?,?,?,?,?);",
			uuid.New().String(),
			new_link.Summary,
			new_link.LinkID,
			req_user_id,
			new_link.SubmitDate,
		); err != nil {
			render.Render(w, r, e.Err500(err))
			return
		} else {
			new_link.SummaryCount += 1
		}
	}

	// Insert tag
	new_link.Cats = util.AlphabetizeCats(request.Cats)
	if _, err = tx.Exec(
		"INSERT INTO Tags VALUES(?,?,?,?,?);",
		uuid.New().String(),
		new_link.LinkID,
		new_link.Cats,
		new_link.SubmittedBy,
		new_link.SubmitDate,
	); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	// Insert link
	new_link.URL = final_url
	if new_link.Summary == "" && new_link.AutoSummary != "" {
		new_link.Summary = new_link.AutoSummary
	}
	new_link.PreviewImgFilename = util.SavePreviewImgAndGetFileName(
		new_link.PreviewImgURL,
		new_link.LinkID,
	)

	if _, err = tx.Exec(
		"INSERT INTO Links VALUES(?,?,?,?,?,?,?);",
		new_link.LinkID,
		new_link.URL,
		new_link.SubmittedBy,
		new_link.SubmitDate,
		new_link.Cats,
		new_link.Summary,
		new_link.PreviewImgFilename,
	); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	// Increment spellfix ranks
	if err = util.IncrementSpellfixRanksForCats(
		tx,
		strings.Split(request.Cats, ","),
	); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	if err = tx.Commit(); err != nil {
		render.Render(w, r, e.Err500(err))
		return
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

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]any)["login_name"].(string)
	if !util.UserSubmittedLink(req_login_name, request.LinkID) {
		render.Render(w, r, e.ErrUnauthorized(e.ErrDoesntOwnLink))
		return
	}

	// Fetch global cats before deleting so spellfix ranks can be updated
	var gc string
	err = db.Client.QueryRow("SELECT global_cats FROM Links WHERE id = ?;", request.LinkID).Scan(&gc)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	tx, err := db.Client.Begin()
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}
	defer tx.Rollback()

	if _, err = tx.Exec(
		"DELETE FROM Links WHERE id = ?;",
		request.LinkID,
	); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	if err = util.DecrementSpellfixRanksForCats(
		tx,
		strings.Split(gc, ","),
	); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

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

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]any)["login_name"].(string)
	if util.UserSubmittedLink(req_login_name, link_id) {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrCannotLikeOwnLink))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	if util.UserHasLikedLink(req_user_id, link_id) {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrLinkAlreadyLiked))
		return
	}

	new_like_id := uuid.New().String()
	_, err := db.Client.Exec(
		`INSERT INTO "Link Likes" VALUES(?,?,?,?);`,
		new_like_id,
		link_id,
		req_user_id,
		mutil.NEW_LONG_TIMESTAMP(),
		
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

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
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

	req_login_name := r.Context().Value(m.JWTClaimsKey).(map[string]any)["login_name"].(string)
	owns_link := util.UserSubmittedLink(req_login_name, link_id)
	if owns_link {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrCannotCopyOwnLink))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	already_copied := util.UserHasCopiedLink(req_user_id, link_id)
	if already_copied {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrLinkAlreadyCopied))
		return
	}

	new_copy_id := uuid.New().String()

	_, err := db.Client.Exec(
		`INSERT INTO "Link Copies" VALUES(?,?,?,?);`,
		new_copy_id,
		link_id,
		req_user_id,
		mutil.NEW_LONG_TIMESTAMP(),
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

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	already_copied := util.UserHasCopiedLink(req_user_id, link_id)
	if !already_copied {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrLinkNotCopied))
		return
	}

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

func ClickLink(w http.ResponseWriter, r *http.Request) {
	request := &model.NewClickRequest{}
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	link_exists, err := util.LinkExists(request.LinkID)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	} else if !link_exists {
		render.Render(w, r, e.ErrUnprocessable(e.ErrNoLinkWithID))
		return
	}

	result := struct {
		ID        string `json:"id"`
		LinkID    string `json:"link_id"`
		UserID    string `json:"user_id"`
		IPAddr    string `json:"ip_addr"`
		Timestamp string `json:"timestamp"`
	}{
		LinkID:    request.LinkID,
		Timestamp: request.Timestamp,
	}

	// Get user ID, or IP address if not signed in
	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	if req_user_id == "" {
		if r.RemoteAddr == "" {
			render.Render(w, r, e.ErrInvalidRequest(e.ErrNoUserOrIP))
			return
		}

		result.UserID = "anonymous"
		result.IPAddr = r.RemoteAddr
	} else {
		result.UserID = req_user_id
	}

	result.ID = uuid.New().String()

	if _, err = db.Client.Exec(
		`INSERT INTO "Clicks" VALUES(?,?,?,?,?);`,
		result.ID,
		result.LinkID,
		result.UserID,
		result.IPAddr,
		result.Timestamp,
	); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, result)
}
