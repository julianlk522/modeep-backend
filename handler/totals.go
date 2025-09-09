package handler

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/julianlk522/modeep/db"
	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func GetTotals(w http.ResponseWriter, r *http.Request) {
	totals_sql := query.NewTotals()
	var totals model.Totals

	if err := db.Client.
		QueryRow(
			totals_sql.Text, 
			totals_sql.Args...,
		).
		Scan(&totals.Links,
			&totals.Clicks,
			&totals.Contributors,
			&totals.LinksStarred,
			&totals.Tags,
			&totals.Summaries,
		); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, totals)
}
