package manifestro

import "github.com/cloudfoundry-incubator/candiedyaml"

type manifestYaml struct {
	Applications []struct {
		Instances *int
	}
}

func GetInstances(manifest string) *int {
	var m manifestYaml

	err := candiedyaml.Unmarshal([]byte(manifest), &m)
	if err != nil || m.Applications[0].Instances == nil || *m.Applications[0].Instances < 1 {
		return nil
	}

	return m.Applications[0].Instances
}
