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

func (s *SignUpRequest) Bind(r *http.Request) error {
	switch {
	case s.Auth.LoginName == "":
		return e.ErrNoLoginName
	case len(s.Auth.LoginName) < util.LOGIN_NAME_LOWER_LIMIT:
		return e.LoginNameExceedsLowerLimit(util.LOGIN_NAME_LOWER_LIMIT)
	case len(s.Auth.LoginName) > util.LOGIN_NAME_UPPER_LIMIT:
		return e.LoginNameExceedsUpperLimit(util.LOGIN_NAME_UPPER_LIMIT)
	case util.ContainsInvalidChars(s.Auth.LoginName):
		return e.ErrLoginNameContainsInvalidChars

	case s.Auth.Password == "":
		return e.ErrNoPassword
	case len(s.Auth.Password) < util.PASSWORD_LOWER_LIMIT:
		return e.PasswordExceedsLowerLimit(util.PASSWORD_LOWER_LIMIT)
	case len(s.Auth.Password) > util.PASSWORD_UPPER_LIMIT:
		return e.PasswordExceedsUpperLimit(util.PASSWORD_UPPER_LIMIT)
	}

	s.ID = uuid.New().String()
	s.CreatedAt = util.NEW_SHORT_TIMESTAMP()
	return nil
}

type LogInRequest struct {
	*Auth
}

func (l *LogInRequest) Bind(r *http.Request) error {
	if l.Auth.LoginName == "" {
		return e.ErrNoLoginName
	} else if l.Auth.Password == "" {
		return e.ErrNoPassword
	}

	return nil
}

// PROFILE
type Profile struct {
	LoginName string
	About     string
	PFP       string
	Created   string
}

type EditAboutRequest struct {
	About string `json:"about"`
}

func (ea *EditAboutRequest) Bind(r *http.Request) error {
	if len(ea.About) > util.PROFILE_ABOUT_CHAR_LIMIT {
		return e.ProfileAboutLengthExceedsLimit(util.PROFILE_ABOUT_CHAR_LIMIT)
	} else if len(ea.About) > 0 && !regexp.MustCompile(`[^\n\r\s\p{C}]`).MatchString(ea.About) {
		return e.ErrAboutHasInvalidChars
	}

	return nil

}

type EditProfilePicRequest struct {
	ProfilePic string `json:"pfp,omitempty"`
}

// TREASURE MAP
type TmapSections[T TmapLink | TmapLinkSignedIn] struct {
	Cats      *[]CatCount
	Submitted *[]T
	Tagged    *[]T
	Copied    *[]T
}

type Tmap[T TmapLink | TmapLinkSignedIn] struct {
	Profile *Profile
	NSFWLinksCount int
	*TmapSections[T]
}

type FilteredTmap[T TmapLink | TmapLinkSignedIn] struct {
	NSFWLinksCount int
	*TmapSections[T]
}

type TmapCatCountsOpts struct {
	OmittedCats []string
}
