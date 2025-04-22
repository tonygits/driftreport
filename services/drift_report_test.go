package services

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/driftreport/entities"
	"github.com/driftreport/mocks"
	"github.com/driftreport/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDriftReportService(t *testing.T) {
	awsProvider := mocks.NewAWSProvider()
	driftSvc := NewDriftReportService(awsProvider)
	logger := utils.InitZapLog()
	defer logger.Sync() // Flush any buffered log messages

	ctx := context.Background()

	ctx1, cancel1 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel1()

	Convey("load terraform instances from terraform.tfstate.json file", t, func() {
		_, instanceIds, err := loadTerraformStateInstances("../terraform.tfstate.json")
		So(err, ShouldBeNil)
		So(len(instanceIds), ShouldEqual, 1)
	})

	Convey("Test parsing an empty .tfstate file", t, func() {
		var buffer bytes.Buffer
		buffer.WriteString("")
		content, err := io.ReadAll(&buffer)
		So(err, ShouldBeNil)
		err = os.WriteFile("../tfstate.json", content, 0644)
		So(err, ShouldBeNil)

		_, _, err = loadTerraformStateInstances("../tfstate.json")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "failed with code 400: .tfstate is empty")
	})

	Convey("test print drift report tabular format if drifted", t, func() {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		driftReports := []*entities.DriftReport{
			{
				InstanceID: "rhhejbdjenfr",
				Drifted: true,
				Differences: map[string]string{
					"security_groups":"AWS: [sg-091fde8327f3fe99a], Terraform: [example-security-group]",
				},
			},
		}
		printDriftTable(driftReports)
		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout
		So(string(out), ShouldEqual, "INSTANCE ID    |DRIFTED   |ATTRIBUTES WITH DIFFERENCES\nrhhejbdjenfr   |true      |security_groups: AWS: [sg-091fde8327f3fe99a], Terraform: [example-security-group]\n")
	})

	Convey("test print drift report tabular format if not drifted", t, func() {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		driftReports := []*entities.DriftReport{
			{
				InstanceID: "rhhejbdjenfr",
				Drifted: false,
				Differences: map[string]string{
					"security_groups":"AWS: [sg-091fde8327f3fe99a], Terraform: [sg-091fde8327f3fe99a]",
				},
			},
		}
		printDriftTable(driftReports)
		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout
		So(string(out), ShouldEqual, "INSTANCE ID    |DRIFTED   |ATTRIBUTES WITH DIFFERENCES\nrhhejbdjenfr   |false     |No differences\n")
	})

	Convey("general drift report with no attributes", t, func() {
		tfInstanceMap, instanceIds, err := loadTerraformStateInstances("../terraform.tfstate.json")
		So(err, ShouldBeNil)
		So(len(instanceIds), ShouldEqual, 1)

		attributesList := "instance_type,security_groups,tags"
		attributes := make(map[string]bool)
		for _, attr := range strings.Split(attributesList, ",") {
			attributes[attr] = false
		}
		_, err = driftChecker(instanceIds[0], nil, tfInstanceMap[instanceIds[0]], attributes)
		So(err, ShouldNotBeNil)
	})

	Convey("general drift report with attributes", t, func() {
		tfInstanceMap, instanceIds, err := loadTerraformStateInstances("../terraform.tfstate.json")
		So(err, ShouldBeNil)
		So(len(instanceIds), ShouldEqual, 1)

		attributesList := "instance_type,security_groups,tags"
		attributes := make(map[string]bool)
		for _, attr := range strings.Split(attributesList, ",") {
			attributes[attr] = true
		}
		_, err = driftChecker(instanceIds[0], nil, tfInstanceMap[instanceIds[0]], attributes)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "failed with code 400: ec2 instance not set")

		_, err = driftChecker(instanceIds[0], &entities.EC2Instance{}, nil, attributes)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "failed with code 400: terraform instance not set")
	})

	Convey("print drift report within context deadline ", t, func() {
		err := driftSvc.PrintDriftReport(ctx1)
		So(err, ShouldBeNil)
	})
}
