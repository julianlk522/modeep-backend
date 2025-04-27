package handler

import (
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/google/uuid"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/handler/util"
	m "github.com/julianlk522/fitm/middleware"
	"github.com/julianlk522/fitm/model"
)

var profile_pic_dir string

func init() {
	work_dir, _ := os.Getwd()
	profile_pic_dir = filepath.Join(work_dir, "db/img/profile")
}

func EditAbout(w http.ResponseWriter, r *http.Request) {
	edit_about_data := &model.EditAboutRequest{}
	if err := render.Bind(r, edit_about_data); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	_, err := db.Client.Exec(
		`UPDATE Users SET about = ? WHERE id = ?`,
		edit_about_data.About,
		req_user_id,
	)
	if err != nil {
		log.Fatal(err)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, edit_about_data)
}

func GetProfilePic(w http.ResponseWriter, r *http.Request) {
	var file_name string = chi.URLParam(r, "file_name")
	path := profile_pic_dir + "/" + file_name

	if _, err := os.Stat(path); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrProfilePicNotFound))
		return
	}

	http.ServeFile(w, r, path)
}

func UploadProfilePic(w http.ResponseWriter, r *http.Request) {
	// Get file (up to 10MB, or 10 * 2^20 bytes)
	r.ParseMultipartForm(10 << 20)
	file, handler, err := r.FormFile("pic")
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}
	defer file.Close()

	if !strings.Contains(handler.Header.Get("Content-Type"), "image") {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrInvalidFileType))
		return
	}

	img, _, err := image.Decode(file)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	if !util.HasAcceptableAspectRatio(img) {
		render.Render(
			w, r, e.ErrInvalidRequest(e.ErrInvalidProfilePicAspectRatio),
		)
		return
	}

	if _, err := os.Stat(profile_pic_dir); err != nil {
		render.Render(w, r, e.Err500(e.ErrCouldNotCreateProfilePic))
		return
	}

	extension := filepath.Ext(handler.Filename)
	unique_name := uuid.New().String() + extension
	full_path := profile_pic_dir + "/" + unique_name

	dst, err := os.Create(full_path)
	if err != nil {
		render.Render(w, r, e.Err500(e.ErrCouldNotCreateProfilePic))
		return
	}
	defer dst.Close()

	file.Seek(0, 0)

	if _, err := io.Copy(dst, file); err != nil {
		render.Render(w, r, e.Err500(e.ErrCouldNotCopyProfilePic))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	_, err = db.Client.Exec(`UPDATE Users SET pfp = ? WHERE id = ?`, unique_name, req_user_id)
	if err != nil {
		render.Render(w, r, e.Err500(e.ErrCouldNotSaveProfilePic))
		return
	}

	http.ServeFile(w, r, full_path)
}

func DeleteProfilePic(w http.ResponseWriter, r *http.Request) {
	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)

	if has_pfp := util.UserWithIDHasProfilePic(req_user_id); !has_pfp {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoProfilePic))
		return
	}

	// Get file path before deleting
	var pfp string
	err := db.Client.QueryRow(`SELECT pfp FROM Users WHERE id = ?`, req_user_id).Scan(&pfp)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}
	pfp_path := profile_pic_dir + "/" + pfp

	// Delete from DB
	_, err = db.Client.Exec(
		`UPDATE Users SET pfp = NULL WHERE id = ?`,
		req_user_id,
	)
	if err != nil {
		render.Render(w, r, e.Err500(e.ErrCouldNotRemoveProfilePic))
		return
	}

	if _, err := os.Stat(pfp_path); err == nil {
		err = os.Remove(pfp_path)
		if err != nil {
			render.Render(w, r, e.Err500(e.ErrCouldNotRemoveProfilePic))
			return
		}
	} else {
		log.Print("pfp was not present on filesystem at saved path")
	}

	w.WriteHeader(http.StatusNoContent)
}

func GetTreasureMap(w http.ResponseWriter, r *http.Request) {
	var login_name string = chi.URLParam(r, "login_name")
	if login_name == "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLoginName))
		return
	}

	user_exists, err := util.UserExists(login_name)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	} else if !user_exists {
		render.Render(w, r, e.Err404(e.ErrNoUserWithLoginName))
		return
	}

	opts, err := util.GetTmapOptsFromRequestParams(
		r.URL.Query(),
	)
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}
	opts.OwnerLoginName = login_name

	var tmap any

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
	if req_user_id != "" {
		opts.AsSignedInUser = req_user_id
		tmap, err = util.BuildTmapFromOpts[model.TmapLinkSignedIn](opts)
	} else {
		tmap, err = util.BuildTmapFromOpts[model.TmapLink](opts)
	}

	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	render.JSON(w, r, tmap)
}
