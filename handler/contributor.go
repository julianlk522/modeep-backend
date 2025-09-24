package handler

import (
	"net/http"

	"github.com/go-chi/render"
	e "github.com/julianlk522/modeep/error"
	util "github.com/julianlk522/modeep/handler/util"
	"github.com/julianlk522/modeep/query"
)

func GetTopContributors(w http.ResponseWriter, r *http.Request) {
	contributors_sql := query.
		NewTopContributors().
		FromRequestParams(
			r.URL.Query(),
		)

	if contributors_sql.Error != nil {
		render.Render(w, r, e.ErrInvalidRequest(contributors_sql.Error))
		return
	}

	contributors := util.ScanContributors(contributors_sql)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, contributors)
}
