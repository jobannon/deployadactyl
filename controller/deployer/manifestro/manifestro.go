package manifestro

import "github.com/cloudfoundry-incubator/candiedyaml"

type manifestYaml struct {
	Applications []struct {
		Instances *uint16
	}
}

func GetInstances(manifest string) *uint16 {
	var m manifestYaml

	err := candiedyaml.Unmarshal([]byte(manifest), &m)
	if err != nil || m.Applications[0].Instances == nil || *m.Applications[0].Instances < 1 {
		return nil
	}

	return m.Applications[0].Instances
}
