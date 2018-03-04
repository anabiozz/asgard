package inputs

import (
	"heimdall_project/asgard"
)

// Creator ...
type Creator func() asgard.Input

// Inputs ...
var Inputs = map[string]Creator{}

// Add function - In main file connected all inputs plugin.
// Add func is called In plugin in init function adding plugin in Inputs variable
func Add(name string, creator Creator) {
	Inputs[name] = creator
}
