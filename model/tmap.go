package model

import (
	"net/http"
	"regexp"

	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/model/util"
)

// Profile
type Profile struct {
	LoginName string
	About     string
	PFP       string
	Created   string
}

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

// Links
type TmapLink struct {
	Link
	CatsFromUser bool
}

type TmapLinkSignedIn struct {
	LinkSignedIn
	CatsFromUser bool
}

type TmapSections[T TmapLink | TmapLinkSignedIn] struct {
	Submitted        *[]T
	Tagged           *[]T
	Copied           *[]T
	SectionsWithMore []string
	Cats             *[]CatCount
}

type Tmap[T TmapLink | TmapLinkSignedIn] struct {
	*TmapSections[T]
	NSFWLinksCount int
	Profile        *Profile
}

type FilteredTmap[T TmapLink | TmapLinkSignedIn] struct {
	*TmapSections[T]
	NSFWLinksCount int
}

type PaginatedTmapSection[T TmapLink | TmapLinkSignedIn] struct {
	Links          *[]T
	Cats           *[]CatCount
	NSFWLinksCount int
	NextPage       int
}

type TmapCatCountsOptions struct {
	RawCatsParams string
}

type TmapOptions struct {
	OwnerLoginName string
	// RawCatsParams (reserved chars unescaped, plural/singular variations not
	// bundled) is stored in addition to CatsFilter so that
	// GetCatCountsFromTmapLinks can know the exact values passed in
	// the request and not count them
	RawCatsParams  string
	CatsFilter     []string
	AsSignedInUser string
	SortByNewest   bool
	IncludeNSFW    bool
	Section        string
	Page           int
}