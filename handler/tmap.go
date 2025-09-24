package handler

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/google/uuid"

	"github.com/julianlk522/modeep/db"
	e "github.com/julianlk522/modeep/error"
	util "github.com/julianlk522/modeep/handler/util"
	m "github.com/julianlk522/modeep/middleware"
	"github.com/julianlk522/modeep/model"
)

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
	path := util.Profile_pic_dir + "/" + file_name

	if _, err := os.Stat(path); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrProfilePicNotFound))
		return
	}

	http.ServeFile(w, r, path)
}

func UploadProfilePic(w http.ResponseWriter, r *http.Request) {
	// Get file (up to 10MB, or 10 * 2^20 bytes)
	r.ParseMultipartForm(10 << 20)

	// Verify valid
	pic_file_bytes, handler, err := r.FormFile("pic")
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}
	defer pic_file_bytes.Close()

	if !strings.Contains(handler.Header.Get("Content-Type"), "image") {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrInvalidFileType))
		return
	}

	// Accepted; save file
	upload := &model.ImgUpload{
		Bytes: pic_file_bytes,
		Purpose: "ProfilePic",
		UID: uuid.New().String(),
	}

	var file_name string
	if file_name, err = util.SaveUploadedImg(upload); err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	req_user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)

	// Delete old pic if there was one
	if has_pfp := util.UserWithIDHasProfilePic(req_user_id); has_pfp {
		var current_file_name string
		if err := db.Client.QueryRow(`SELECT pfp FROM Users WHERE id = ?`, req_user_id).Scan(&current_file_name); err != nil {
			render.Render(w, r, e.ErrInternalServerError(err))
			return
		}
		
		pfp_path := util.Profile_pic_dir + "/" + current_file_name
		if err = os.Remove(pfp_path); err != nil {
			log.Printf("Could not remove old profile pic: %s", err)
		}
	}

	// Update DB
	if _, err = db.Client.Exec(
		`UPDATE Users SET pfp = ? WHERE id = ?`, 
		file_name, 
		req_user_id,
	); err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	
	file_path := util.Profile_pic_dir + "/" + file_name
	http.ServeFile(w, r, file_path)
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
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}
	pfp_path := util.Profile_pic_dir + "/" + pfp

	// Delete from DB
	if _, err = db.Client.Exec(
		`UPDATE Users SET pfp = NULL WHERE id = ?`,
		req_user_id,
	); err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	if _, err := os.Stat(pfp_path); err == nil {
		err = os.Remove(pfp_path)
		if err != nil {
			render.Render(w, r, e.ErrInternalServerError(e.ErrCouldNotDeleteProfilePicFile))
			return
		}
	} else {
		log.Print("Profile pic was not present on filesystem at saved path")
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
		render.Render(w, r, e.ErrNotFound(e.ErrNoUserWithLoginName))
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
