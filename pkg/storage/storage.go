package storage

import (
	"github.com/pkg/errors"
	"github.com/redhat-developer/ocdev/pkg/occlient"
)

func Add(config *occlient.VolumeConfig) (string, error) {
	oc := occlient.Oc{}

	output, err := oc.SetVolumes(config,
		&occlient.VolumeOperations{
			Add: true,
		})
	if err != nil {
		return "", errors.Wrap(err, "unable to create storage")
	}
	return output, nil
}

func Remove(config *occlient.VolumeConfig) (string, error) {
	oc := occlient.Oc{}

	output, err := oc.SetVolumes(config,
		&occlient.VolumeOperations{
			Remove: true,
		})
	if err != nil {
		return "", errors.Wrap(err, "unable to remove storage")
	}
	return output, nil
}

func List(config *occlient.VolumeConfig) (string, error) {
	oc := occlient.Oc{}
	output, err := oc.SetVolumes(config,
		&occlient.VolumeOperations{
			List: true,
		})
	if err != nil {
		return "", errors.Wrap(err, "unable to list storage")
	}
	return output, nil
}
