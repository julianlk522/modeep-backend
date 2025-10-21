package model

// PERIODS
// Valid: day, week, month, year, all
type Period string

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
	PeriodYear  Period = "year"
	PeriodAll   Period = "all"
)

var ValidPeriodsInDays = map[Period]uint16{
	PeriodDay:   1,
	PeriodWeek:  7,
	PeriodMonth: 30,
	PeriodYear:  365,
	PeriodAll:   0,
}

// SORT BY
// Valid: times_starred, avg_stars, newest, oldest, clicks
type SortBy string

const (
	SortByTimesStarred SortBy = "times_starred"
	SortByAverageStars SortBy = "avg_stars"
	SortByNewest       SortBy = "newest"
	SortByOldest       SortBy = "oldest"
	SortByClicks       SortBy = "clicks"
)

var ValidSortBys = [5]SortBy{
	SortByTimesStarred,
	SortByAverageStars,
	SortByNewest,
	SortByOldest,
	SortByClicks,
}

// TREASURE MAP SECTION
// Valid: submitted, starred, tagged
type TmapIndividualSectionName string

const (
	TmapSectionSubmitted TmapIndividualSectionName = "submitted"
	TmapSectionStarred   TmapIndividualSectionName = "starred"
	TmapSectionTagged    TmapIndividualSectionName = "tagged"
)

var ValidTmapSections = [3]TmapIndividualSectionName{
	TmapSectionSubmitted,
	TmapSectionStarred,
	TmapSectionTagged,
}

