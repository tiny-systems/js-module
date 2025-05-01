package lib

import (
	"context"
	"github.com/grafana/sobek"
)

// Exports is representation of ESM exports of a module
type Exports struct {
	// Default is what will be the `default` export of a module
	Default interface{}
	// Named is the named exports of a module
	Named map[string]interface{}
}

// Module is the interface js modules should implement in order to get access to the VU
type Module interface {
	// NewModuleInstance will get modules.VU that should provide the module with a way to interact with the VU.
	// This method will be called for *each* VU that imports the module *once* per that VU.
	NewModuleInstance(VU) Instance
}

// Instance is what a module needs to return
type Instance interface {
	Exports() Exports
}

// VU gives access to the currently executing VU to a module Instance
type VU interface {
	// Context return the context.Context about the current VU
	Context() context.Context

	// Runtime returns the sobek.Runtime for the current VU
	Runtime() *sobek.Runtime
}
