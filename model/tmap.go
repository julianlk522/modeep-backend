package model

import (
	"net/http"
	"regexp"

	e "github.com/julianlk522/modeep/error"
	util "github.com/julianlk522/modeep/model/util"
)

// OPTIONS
type TmapOptions struct {
	OwnerLoginName string
	// RawCatFiltersParams (reserved chars unescaped, plural/singular variations not
	// bundled) is stored in addition to CatFilters so that
	// GetCatCountsFromTmapLinks() can know the exact values passed in
	// the request and not count them
	RawCatFiltersParams                    string
	CatFiltersWithSpellingVariants         []string
	NeuteredCatFiltersWithSpellingVariants []string
	AsSignedInUser                         string
	IncludeNSFW                            bool
	SortBy                                 SortBy
	Period                                 Period
	SummaryContains                        string
	URLContains                            string
	URLLacks                               string
	Section                                TmapIndividualSectionName
	Page                                   int
}

type TmapNSFWLinksCountOptions struct {
	Section                                TmapIndividualSectionName
	CatFiltersWithSpellingVariants         []string
	NeuteredCatFiltersWithSpellingVariants []string
	Period                                 Period
	SummaryContains                        string
	URLContains                            string
	URLLacks                               string
}

type TmapCatCountsOptions struct {
	RawCatsParams string
}

// LINKS
type TmapLink struct {
	Link
	CatsFromUser bool
}

type TmapLinkSignedIn struct {
	LinkSignedIn
	CatsFromUser bool
}

// SECTIONS
type TmapSections[T TmapLink | TmapLinkSignedIn] struct {
	Submitted        *[]T
	Starred          *[]T
	Tagged           *[]T
	SectionsWithMore []string
	Cats             *[]CatCount
}

// Individual section of Treasure Map links:
// (submitted, starred, or tagged)
type TmapIndividualSectionPage[T TmapLink | TmapLinkSignedIn] struct {
	Links          *[]T
	Cats           *[]CatCount
	NSFWLinksCount int
	// Individual sections can be paginated for thorough searches,
	// though main Treasure Map page just has the first few links
	// from each section as an overview
	Pages int
}

type TmapIndividualSectionWithCatFiltersPage[T TmapLink | TmapLinkSignedIn] struct {
	*TmapIndividualSectionPage[T]
	MergedCats []string
}

type TmapPage[T TmapLink | TmapLinkSignedIn] struct {
	*TmapSections[T]
	NSFWLinksCount int
}

type TmapWithCatFiltersPage[T TmapLink | TmapLinkSignedIn] struct {
	*TmapPage[T]
	MergedCats []string
}

type TmapWithProfilePage[T TmapLink | TmapLinkSignedIn] struct {
	*TmapPage[T]
	// Profile data does not need to be viewed on every single page of
	// someone's Treasure Map, but it available on the "blank slate" version:
	// that is, when no cat filters are applied
	Profile *Profile
}

type Profile struct {
	LoginName string
	PFP       string
	About     string
	Email     string
	CreatedAt string
}

// REQUESTS
type EditAboutRequest struct {
	About string `json:"about"`
}

func (ear *EditAboutRequest) Bind(r *http.Request) error {
	if len(ear.About) > util.PROFILE_ABOUT_CHAR_LIMIT {
		return e.ProfileAboutLengthExceedsLimit(util.PROFILE_ABOUT_CHAR_LIMIT)

		// cannot have ONLY newline, carriage return, whitespace, or
		// unicode control characters
		// though they are allowed if there is other text
	} else if len(ear.About) > 0 && !regexp.MustCompile(`[^\n\r\s\p{C}]`).MatchString(ear.About) {
		return e.ErrAboutHasInvalidChars
	}

	return nil

}

type EditProfilePicRequest struct {
	ProfilePic string `json:"pfp,omitempty"`
}
