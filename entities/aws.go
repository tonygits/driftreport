package entities

type (
	EC2Instance struct {
		InstanceType   string            `json:"instance_type"`
		SecurityGroups []string          `json:"security_groups"`
		Tags           map[string]string `json:"tags"`
	}

	TerraformState struct {
		Resources []struct {
			Mode      string      `json:"mode"`
			Type      string      `json:"type"`
			Name      string      `json:"name"`
			Instances []*Instance `json:"instances"`
		} `json:"resources"`
	}

	Instance struct {
		Attributes struct {
			InstanceID     string            `json:"id"`
			Type           string            `json:"instance_type"`
			Tags           map[string]string `json:"tags"`
			State          string            `json:"instance_state"`
			SecurityGroups []string          `json:"security_groups"`
		} `json:"attributes"`
	}

	DriftReport struct {
		InstanceID  string            `json:"instance_id"`
		Drifted     bool              `json:"drifted"`
		Differences map[string]string `json:"differences"`
	}
)
