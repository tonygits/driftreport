package entities

import "fmt"

type CustomError struct {
	StatusCode int
	Err        error
}

func (c *CustomError) Error() string {
	return fmt.Sprintf("failed with code %d: %s", c.StatusCode, c.Err)
}
