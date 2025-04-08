package entities

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func doSomething() error {
	return &CustomError{
		StatusCode: 404,
		Err:        errors.New("resource not found"),
	}
}

func TestCustomError(t *testing.T) {
	Convey("Test custom error", t, func() {
		err := doSomething()
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "failed with code 404: resource not found")
	})
}
