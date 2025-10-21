package handler

import (
	"log"
	"net/url"
	"strings"

	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func GetTopContributorsOptionsFromRequestParams(params url.Values) (*model.TopContributorsOptions, error) {
	var opts = &model.TopContributorsOptions{}
	
	cat_filters_params := params.Get("cats")
	if cat_filters_params != "" {
		opts.CatFiltersWithSpellingVariants = query.GetCatsOptionalPluralOrSingularForms(
			strings.Split(cat_filters_params, ","),
		)
	}
	neutered_params := params.Get("neutered")
	if neutered_params != "" {
		// Since we use IN, not FTS MATCH, spelling variants are not
		// needed (and casing matters)
		opts.NeuteredCatFilters = strings.Split(neutered_params, ",")
	}
	summary_contains_params := params.Get("summary_contains")
	if summary_contains_params != "" {
		opts.SummaryContains = summary_contains_params
	}
	url_contains_params := params.Get("url_contains")
	if url_contains_params != "" {
		opts.URLContains = url_contains_params
	}
	url_lacks_params := params.Get("url_lacks")
	if url_lacks_params != "" {
		opts.URLLacks = url_lacks_params
	}
	period_params := params.Get("period")
	if period_params != "" {
		period := model.Period(period_params)
		if _, ok := model.ValidPeriodsInDays[period]; ok {
			opts.Period = period
		} else {
			return nil, e.ErrInvalidPeriod
		}
	}

	return opts, nil
}

func ScanContributors(contributors_sql *query.Contributors) *[]model.Contributor {
	rows, err := contributors_sql.ValidateAndExecuteRows()
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
