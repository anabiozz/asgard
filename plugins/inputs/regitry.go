package inputs

import (
	"heimdall_project/asgard"
)

type Creator func() asgard.Input

var Inputs = map[string]Creator{}

func Add(name string, creator Creator) {
	Inputs[name] = creator
}
