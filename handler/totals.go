package handler

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

func GetTotals(w http.ResponseWriter, r *http.Request) {
	totals_sql := query.NewTotals()
	var totals model.Totals

	if err := db.Client.
		QueryRow(totals_sql.Text).
		Scan(&totals.Links, 
			&totals.Clicks, 
			&totals.Contributors,
			&totals.Likes, 
			&totals.Tags, 
			&totals.Summaries, 
		); err != nil {
			render.Render(w, r, e.Err500(err))
			return
		}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, totals)
}

