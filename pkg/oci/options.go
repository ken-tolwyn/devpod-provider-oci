package oci

import (
	"fmt"
	"os"
	"strings"
)

var (
	OCI_TENANCY_OCID     = "OCI_TENANCY_OCID"
	OCI_USER_OCID        = "OCI_USER_OCID"
	OCI_REGION           = "OCI_REGION"
	OCI_COMPARTMENT_OCID = "OCI_COMPARTMENT_OCID"
	OCI_VCN_OCID         = "OCI_VCN_OCID"
	OCI_SUBNET_OCID      = "OCI_SUBNET_OCID"
	OCI_IMAGE_OCID       = "OCI_IMAGE_OCID"
	OCI_SHAPE            = "OCI_SHAPE"
	OCI_SSH_PUBLIC_KEY   = "OCI_SSH_PUBLIC_KEY"
	OCI_PRIVATE_KEY_PATH = "OCI_PRIVATE_KEY_PATH"
	OCI_FINGERPRINT      = "OCI_FINGERPRINT"
)

type Options struct {
	TenancyOCID     string
	UserOCID        string
	Region          string
	CompartmentOCID string
	VCNOCID         string
	SubnetOCID      string
	ImageOCID       string
	Shape           string
	SSHPublicKey    string
	PrivateKeyPath  string
	Fingerprint     string

	MachineFolder string
	MachineID     string
}

func FromEnv(withFolder bool) (*Options, error) {
	retOptions := &Options{}

	var err error
	retOptions.TenancyOCID, err = fromEnvOrError(OCI_TENANCY_OCID)
	if err != nil {
		return nil, err
	}
	retOptions.UserOCID, err = fromEnvOrError(OCI_USER_OCID)
	if err != nil {
		return nil, err
	}
	retOptions.Region, err = fromEnvOrError(OCI_REGION)
	if err != nil {
		return nil, err
	}
	retOptions.CompartmentOCID, err = fromEnvOrError(OCI_COMPARTMENT_OCID)
	if err != nil {
		return nil, err
	}
	retOptions.VCNOCID = os.Getenv(OCI_VCN_OCID)
	retOptions.SubnetOCID = os.Getenv(OCI_SUBNET_OCID)
	retOptions.ImageOCID = os.Getenv(OCI_IMAGE_OCID)
	retOptions.Shape = os.Getenv(OCI_SHAPE)
	retOptions.SSHPublicKey = os.Getenv(OCI_SSH_PUBLIC_KEY)
	retOptions.PrivateKeyPath = os.Getenv(OCI_PRIVATE_KEY_PATH)
	retOptions.Fingerprint = os.Getenv(OCI_FINGERPRINT)

	retOptions.MachineID, err = fromEnvOrError("MACHINE_ID")
	if err != nil {
		return nil, err
	}
	retOptions.MachineID = "devpod-" + retOptions.MachineID

	if withFolder {
		retOptions.MachineFolder, err = fromEnvOrError("MACHINE_FOLDER")
		if err != nil {
			return nil, err
		}
	}

	return retOptions, nil
}

func fromEnvOrError(name string) (string, error) {
	val := os.Getenv(name)
	if val == "" {
		return "", fmt.Errorf(
			"couldn't find option %s in environment, please make sure %s is defined",
			name,
			name,
		)
	}
	return val, nil
}
