package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/render"
	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/handler/util"
	"github.com/julianlk522/fitm/query"
)

func GetTopContributors(w http.ResponseWriter, r *http.Request) {
	contributors_sql := query.NewContributors()

	cats_params := r.URL.Query().Get("cats")
	if cats_params != "" {
		cats := strings.Split(cats_params, ",")
		contributors_sql = contributors_sql.FromCats(cats)
	}

	period_params := r.URL.Query().Get("period")
	if period_params != "" {
		contributors_sql = contributors_sql.DuringPeriod(period_params)
	}

	if contributors_sql.Error != nil {
		render.Render(w, r, e.ErrInvalidRequest(contributors_sql.Error))
		return
	}

	contributors := util.ScanContributors(contributors_sql)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, contributors)
}
