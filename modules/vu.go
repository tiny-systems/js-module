package modules

import (
	"context"
	"github.com/grafana/sobek"
	"github.com/tiny-systems/js-module/lib"
)

type ModuleVU struct {
	ctx     context.Context
	runtime *sobek.Runtime
}

func NewModuleVU(ctx context.Context, runtime *sobek.Runtime) *ModuleVU {
	return &ModuleVU{ctx: ctx, runtime: runtime}
}

func (m *ModuleVU) Context() context.Context {
	return m.ctx
}

func (m *ModuleVU) Runtime() *sobek.Runtime {
	return m.runtime
}

var _ lib.VU = (*ModuleVU)(nil)
