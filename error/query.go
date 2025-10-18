package error

import "fmt"

func ErrArgCountDoesNotMatchTextPlaceholders(got int, expected int) error {
	return fmt.Errorf("invalid arg count (got %d, expected %d)", got, expected)
}
