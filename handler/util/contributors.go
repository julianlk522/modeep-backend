package handler

import (
	"log"

	"github.com/julianlk522/fitm/db"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

func ScanContributors(contributors_sql *query.Contributors) *[]model.Contributor {
	rows, err := db.Client.Query(contributors_sql.Text, contributors_sql.Args...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	contributors := []model.Contributor{}
	for rows.Next() {
		contributor := model.Contributor{}
		err := rows.Scan(
			&contributor.LinksSubmitted,
			&contributor.LoginName,
		)
		if err != nil {
			log.Fatal(err)
		}
		contributors = append(contributors, contributor)
	}

	return &contributors
}
