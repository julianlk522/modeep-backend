package error

import (
	"net/http"

	"github.com/go-chi/render"
)

var ErrNotFound = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found."}

type ErrResponse struct {
	Err            error  `json:"-"`
	HTTPStatusCode int    `json:"-"`
	StatusText     string `json:"status"`
	ErrorText      string `json:"error,omitempty"`
}

func (er *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, er.HTTPStatusCode)
	return nil
}

// e.g., malformed JSON
func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrUnauthenticated(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 401,
		StatusText:     "Unauthenticated.",
		ErrorText:      err.Error(),
	}
}

func ErrUnauthorized(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 403,
		StatusText:     "Unauthorized.",
		ErrorText:      err.Error(),
	}
}

func Err404(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 404,
		StatusText:     "Resource not found.",
		ErrorText:      err.Error(),
	}
}

func ErrContentTooLarge(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 413,
		StatusText:     "Content too large.",
		ErrorText:      err.Error(),
	}
}

// "syntactically valid but semantically invalid"
// e.g., nonexistent ID provided
func ErrUnprocessable(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Unprocessable.",
		ErrorText:      err.Error(),
	}
}

func ErrTooManyRequests(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 429,
		StatusText:     "Too many requests.",
		ErrorText:      err.Error(),
	}
}

func Err500(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 500,
		StatusText:     "Server failed to process request.",
		ErrorText:      err.Error(),
	}
}
