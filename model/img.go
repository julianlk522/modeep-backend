package model

import "io"

type ImgUpload struct {
	Bytes io.Reader
	Purpose string // "LinkPreview" or "ProfilePic"
	UID string
}