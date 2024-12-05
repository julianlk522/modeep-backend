package model

import (
	"net/http"
	"regexp"

	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/model/util"

	"github.com/google/uuid"
)

// AUTH
type Auth struct {
	LoginName string `json:"login_name"`
	Password  string `json:"password"`
}
type SignUpRequest struct {
	*Auth
	ID        string
	CreatedAt string
}

func (sr *SignUpRequest) Bind(r *http.Request) error {
	switch {
	case sr.Auth.LoginName == "":
		return e.ErrNoLoginName
	case len(sr.Auth.LoginName) < util.LOGIN_NAME_LOWER_LIMIT:
		return e.LoginNameExceedsLowerLimit(util.LOGIN_NAME_LOWER_LIMIT)
	case len(sr.Auth.LoginName) > util.LOGIN_NAME_UPPER_LIMIT:
		return e.LoginNameExceedsUpperLimit(util.LOGIN_NAME_UPPER_LIMIT)
	case util.ContainsInvalidChars(sr.Auth.LoginName):
		return e.ErrLoginNameContainsInvalidChars

	case sr.Auth.Password == "":
		return e.ErrNoPassword
	case len(sr.Auth.Password) < util.PASSWORD_LOWER_LIMIT:
		return e.PasswordExceedsLowerLimit(util.PASSWORD_LOWER_LIMIT)
	case len(sr.Auth.Password) > util.PASSWORD_UPPER_LIMIT:
		return e.PasswordExceedsUpperLimit(util.PASSWORD_UPPER_LIMIT)
	}

	sr.ID = uuid.New().String()
	sr.CreatedAt = util.NEW_SHORT_TIMESTAMP()
	return nil
}

type LogInRequest struct {
	*Auth
}

func (lr *LogInRequest) Bind(r *http.Request) error {
	if lr.Auth.LoginName == "" {
		return e.ErrNoLoginName
	} else if lr.Auth.Password == "" {
		return e.ErrNoPassword
	}

	return nil
}

// TREASURE MAP
// Links
type TmapOptions struct {
	OwnerLoginName string
	// RawCatsParams (reserved chars unescaped, plural/singular variations not 
	// bundled) is stored in addition to CatsFilter so that
	// GetCatCountsFromTmapLinks can know the exact values passed in 
	// the request and not count them
	RawCatsParams string
	CatsFilter []string
	AsSignedInUser string
	SortByNewest bool
	IncludeNSFW bool
	Section string
	Page int
}
type TmapSections[T TmapLink | TmapLinkSignedIn] struct {
	Submitted *[]T
	Tagged    *[]T
	Copied    *[]T
	SectionsWithMore []string
	Cats      *[]CatCount
}

type Tmap[T TmapLink | TmapLinkSignedIn] struct {
	*TmapSections[T]
	NSFWLinksCount int
	Profile *Profile
}

type FilteredTmap[T TmapLink | TmapLinkSignedIn] struct {
	*TmapSections[T]
	NSFWLinksCount int
}

type PaginatedTmapSection[T TmapLink | TmapLinkSignedIn] struct {
	Links *[]T
	Cats  *[]CatCount
	NSFWLinksCount int
	NextPage int
}

type TmapCatCountsOptions struct {
	RawCatsParams string
}

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
	} else if len(ear.About) > 0 && !regexp.MustCompile(`[^\n\r\s\p{C}]`).MatchString(ear.About) {
		return e.ErrAboutHasInvalidChars
	}

	return nil

}

type EditProfilePicRequest struct {
	ProfilePic string `json:"pfp,omitempty"`
}
