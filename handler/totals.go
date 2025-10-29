package handler

import (
	"net/http"

	"github.com/go-chi/render"
	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func GetTotals(w http.ResponseWriter, r *http.Request) {
	totals_sql := query.NewTotals()
	var totals model.Totals

	row, err := totals_sql.ValidateAndExecuteRow()
	if err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}
	if err = row.Scan(&totals.Links,
		&totals.Clicks,
		&totals.Contributors,
		&totals.LinksStarred,
		&totals.Tags,
		&totals.Summaries,
	); err != nil {
		render.Render(w, r, e.ErrInternalServerError(err))
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, totals)
}
