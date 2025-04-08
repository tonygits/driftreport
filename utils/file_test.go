package utils

import (
	"bytes"
	"io"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoadTerraformFileUtil(t *testing.T) {
	logger := InitZapLog()
	defer logger.Sync() // Flush any buffered log messages

	Convey("get resources from terraform state json", t, func() {
		state, err := ParseTerraformState("../terraform.tfstate.json")
		So(err, ShouldBeNil)
		So(len(state.Resources), ShouldEqual, 1)
	})

	Convey("Reading an empty terraform.tfstate file", t, func() {
		var buffer bytes.Buffer
		buffer.WriteString("")
		content, err := io.ReadAll(&buffer)
		So(err, ShouldBeNil)
		// write the whole body at once on tfstate.json
		err = os.WriteFile("../tfstate.json", content, 0644)
		So(err, ShouldBeNil)

		_, err = ParseTerraformState("../tfstate.json")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "failed with code 400: .tfstate is empty")
	})
}
