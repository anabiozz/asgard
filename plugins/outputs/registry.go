package outputs

import "github.com/anabiozz/asgard"

type Creator func() asgard.Output

var Outputs = map[string]Creator{}

func Add(name string, creator Creator) {
	Outputs[name] = creator
}
