package handler

import (
	"net/http"

	"github.com/go-chi/render"
	e "github.com/julianlk522/modeep/error"
	util "github.com/julianlk522/modeep/handler/util"
	"github.com/julianlk522/modeep/query"
)

func GetTopContributors(w http.ResponseWriter, r *http.Request) {
	opts, err := util.GetTopContributorsOptionsFromRequestParams(r.URL.Query())
	if err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}
	contributors_sql, err := query.
		NewTopContributors().
		FromOptions(opts)
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}
	contributors := util.ScanContributors(contributors_sql)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, contributors)
}
