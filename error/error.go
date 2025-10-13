package error

import (
	"log"
	"net/http"

	"github.com/go-chi/render"
)

type ErrResponse struct {
	Err            error  `json:"-"`
	HTTPStatusCode int    `json:"-"`
	StatusText     string `json:"status"`
	ErrorText      string `json:"error,omitempty"`
}

func (er *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	log.Printf("%s: %s", er.StatusText, er.ErrorText)
	render.Status(r, er.HTTPStatusCode)
	return nil
}

// e.g., malformed JSON
func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest, // 400
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrUnauthorized(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusUnauthorized, // 401
		StatusText:     "Unauthorized.",
		ErrorText:      err.Error(),
	}
}

func ErrForbidden(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusForbidden, // 403
		StatusText:     "Forbidden.",
		ErrorText:      err.Error(),
	}
}

func ErrNotFound(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusNotFound, // 404
		StatusText:     "Resource not found.",
		ErrorText:      err.Error(),
	}
}

func ErrConflict(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusConflict, // 409
		StatusText:     err.Error(),
		ErrorText:      err.Error(),
	}
}

func ErrContentTooLarge(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusRequestEntityTooLarge, // 413
		StatusText:     "Content too large.",
		ErrorText:      err.Error(),
	}
}

// "syntactically valid but semantically invalid"
// e.g., nonexistent ID provided
func ErrUnprocessable(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusUnprocessableEntity, // 422
		StatusText:     "Unprocessable.",
		ErrorText:      err.Error(),
	}
}

func ErrTooManyRequests(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusTooManyRequests, // 429
		StatusText:     "Too many requests.",
		ErrorText:      err.Error(),
	}
}

func ErrInternalServerError(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusInternalServerError, // 500
		StatusText:     "Server failed to process request.",
		ErrorText:      err.Error(),
	}
}
