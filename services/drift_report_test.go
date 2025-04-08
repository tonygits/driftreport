package services

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/driftreport/mocks"
	"github.com/driftreport/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDriftReportService(t *testing.T) {
	awsProvider := mocks.NewAWSProvider()
	driftSvc := NewDriftReport(awsProvider)
	logger := utils.InitZapLog()
	defer logger.Sync() // Flush any buffered log messages

	ctx := context.Background()

	ctx1, cancel1 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel1()

	Convey("load terraform instances from terraform.tfstate.json file", t, func() {
		instances, err := loadTerraformStateInstances("../terraform/terraform.tfstate.json")
		So(err, ShouldBeNil)
		So(len(instances), ShouldEqual, 1)
	})

	Convey("Test parsing an empty .tfstate file", t, func() {
		var buffer bytes.Buffer
		buffer.WriteString("")
		content, err := io.ReadAll(&buffer)
		So(err, ShouldBeNil)
		err = os.WriteFile("../terraform/tfstate.json", content, 0644)
		So(err, ShouldBeNil)

		_, err = loadTerraformStateInstances("../terraform/tfstate.json")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "failed with code 400: .tfstate is empty")
	})

	Convey("general drift report with no attributes", t, func() {
		instances, err := loadTerraformStateInstances("../terraform/terraform.tfstate.json")
		So(err, ShouldBeNil)
		So(len(instances), ShouldEqual, 1)

		attributesList := "instance_type,security_groups,tags"
		attributes := make(map[string]bool)
		for _, attr := range strings.Split(attributesList, ",") {
			attributes[attr] = false
		}
		_, err = driftSvc.DriftChecker(ctx, instances[0].Attributes.InstanceID, instances[0], attributes)
		So(err, ShouldNotBeNil)
	})

	Convey("general drift report with attributes", t, func() {
		instances, err := loadTerraformStateInstances("../terraform/terraform.tfstate.json")
		So(err, ShouldBeNil)
		So(len(instances), ShouldEqual, 1)

		attributesList := "instance_type,security_groups,tags"
		attributes := make(map[string]bool)
		for _, attr := range strings.Split(attributesList, ",") {
			attributes[attr] = true
		}
		report, err := driftSvc.DriftChecker(ctx, instances[0].Attributes.InstanceID, instances[0], attributes)
		So(err, ShouldBeNil)
		So(report.Drifted, ShouldEqual, true)
		So(report.InstanceID, ShouldEqual, "i-0c568478aa8a54807")
	})

	Convey("print drift report within context deadline ", t, func() {
		err := driftSvc.PrintDriftReport(ctx1)
		So(err, ShouldBeNil)
	})
}
