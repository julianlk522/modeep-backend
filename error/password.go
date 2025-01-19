package error

import "errors"

var (
	ErrNoPasswordResetSecretEnv error = errors.New("password reset secret environment variable not found")
	ErrNoPasswordResetToken     error = errors.New("no password reset token found in the request")
	ErrInvalidTokenFormat error = errors.New("invalid token format")
	ErrInvalidTokenSignature error = errors.New("invalid token signature")
	ErrTokenExpired error = errors.New("token expired")
)

func FailedToMarshalPayload(err error) error {
	return errors.New("failed to marshal payload: " + err.Error())
}

func FailedToUnmarshalPayload(err error) error {
	return errors.New("failed to unmarshal payload: " + err.Error())
}

func FailedToDecodePayload(err error) error {
	return errors.New("failed to decode payload: " + err.Error())
}