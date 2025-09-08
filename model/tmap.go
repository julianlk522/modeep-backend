package model

import (
	"net/http"
	"regexp"

	e "github.com/julianlk522/modeep/error"
	util "github.com/julianlk522/modeep/model/util"
)

// Profile
type Profile struct {
	LoginName string
	PFP       string
	About     string
	Email     string
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

type TmapSectionPage[T TmapLink | TmapLinkSignedIn] struct {
	Links          *[]T
	Cats           *[]CatCount
	NSFWLinksCount int
	Pages       int
}

type TmapOptions struct {
	OwnerLoginName string
	AsSignedInUser string
	// RawCatsParams (reserved chars unescaped, plural/singular variations not
	// bundled) is stored in addition to CatsFilter so that
	// GetCatCountsFromTmapLinks can know the exact values passed in
	// the request and not count them
	RawCatsParams  string
	Cats     []string
	Period         string
	SortBy         string
	IncludeNSFW    bool
	URLContains    string
	URLLacks       string
	Section        string
	Page           int
}

type TmapNSFWLinksCountOptions struct {
	OnlySection string // "Submitted", "Copied", "Tagged"
	CatsFilter []string
	Period string
	URLContains string
	URLLacks string
}

type TmapCatCountsOptions struct {
	RawCatsParams string
}
