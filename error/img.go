package error

import (
	"errors"
)

var (
	ErrInvalidImgUploadPurpose error = errors.New("invalid image upload purpose")
	ErrCannotEncodeAsWebp error = errors.New("cannot encode webp to file")
)