package outputs

import "heimdall_project/asgard"

type Creator func() asgard.Output

var Outputs = map[string]Creator{}

func Add(name string, creator Creator) {
	Outputs[name] = creator
}
