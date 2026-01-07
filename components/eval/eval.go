package eval

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/grafana/sobek"
	"github.com/tiny-systems/js-module/lib"
	"github.com/tiny-systems/js-module/modules"
	"github.com/tiny-systems/module/api/v1alpha1"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"testing/fstest"
	"time"
)

const (
	ComponentName = "eval"
	RequestPort   = "request"
	ResponsePort  = "response"
	ErrorPort     = "error"
)

const (
	mainModule    = "main.js"
	defaultExport = "default"
)

var goModules = map[string]lib.Module{}

type Context any
type InputData any
type OutputData any

type Script struct {
	Name    string `json:"name" required:"true" title:"File name" Description:"e.g. utils.js"`
	Content string `json:"content" required:"true" language:"javascript" title:"Javascript code" format:"code"`
}

// ScriptItem to avoid confusion of Script definition generated from Scripts array and from Script property
type ScriptItem Script

type Settings struct {
	EnableErrorPort bool         `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If error happen, error port will emit an error message" tab:"Settings"`
	OutputData      OutputData   `json:"outputData" configurable:"true" title:"Output object" description:"Specify schema and example data of the output" tab:"Settings"`
	Script          Script       `json:"script" required:"true" title:"Script" description:"Full ECMAScript 5.1 support. Experimental ESM support. Please CDN only ESM modules" tab:"Main script"`
	Modules         []ScriptItem `json:"modules" required:"true" title:"Modules" description:"Full ECMAScript 5.1 support. Experimental ESM support. Please CDN only ESM modules." uniqueItems:"true" tab:"Includes"`
}

type Error struct {
	Context Context `json:"context"`
	Error   string  `json:"error"`
}

type Request struct {
	Context   Context   `json:"context,omitempty" configurable:"true" title:"Context" description:"Arbitrary message to be send alongside with rendered content"`
	InputData InputData `json:"inputData,omitempty" configurable:"true" title:"Input data" description:"Input data" prompt:"generate JSON schema"`
}

type Response struct {
	Context    Context    `json:"context"`
	OutputData OutputData `json:"outputData"`
}

type Component struct {
	settings Settings
	handler  sobek.Callable
	runtime  *sobek.Runtime
}

var defaultEngineSettings = Settings{
	Script: Script{
		Name: mainModule,
		Content: `import {lodash} from "https://cdn.jsdelivr.net/npm/@esm-bundle/lodash@4.17.21/+esm";
import {typeOf} from "utils.js";
export default function(inp) {
  return lodash.isFunction(typeOf) + typeOf(inp)
}`,
	},
	Modules: []ScriptItem{
		{
			Name:    "utils.js",
			Content: `export function typeOf(input) {return typeof input}`,
		},
	},
}

func (h *Component) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ComponentName,
		Description: "JS Eval",
		Info:        "Synchronous only javascript evaluation",
		Tags:        []string{"js", "javascript", "engine"},
	}
}

func (h *Component) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) any {

	switch port {
	case v1alpha1.SettingsPort:
		// compile template
		in, ok := msg.(Settings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}

		h.settings = in

		err := h.init(ctx, in)
		if err != nil {
			return err
		}

	case RequestPort:

		in, ok := msg.(Request)
		if !ok {
			return fmt.Errorf("invalid input")
		}

		if h.handler == nil {
			return fmt.Errorf("handler is not initialised")
		}

		res, err := h.handler(sobek.Undefined(), h.runtime.ToValue(in.InputData))
		if err != nil {
			if !h.settings.EnableErrorPort {
				return err
			}
			return handler(ctx, ErrorPort, Error{
				Context: in.Context,
				Error:   err.Error(),
			})
		}

		result := res.Export()

		if pr, ok := result.(*sobek.Promise); ok {
			if pr.State() != sobek.PromiseStateFulfilled {
				spew.Dump(fmt.Errorf("%s", pr.Result().Export()))
				return fmt.Errorf("%s", pr.Result().Export())
			}
			result = pr.Result().Export()
		}

		return handler(ctx, ResponsePort, Response{
			Context:    in.Context,
			OutputData: result,
		})

	default:
		return fmt.Errorf("port %s is not supoprted", port)
	}
	return nil
}

func (h *Component) init(ctx context.Context, s Settings) error {

	if s.Script.Content == "" {
		return fmt.Errorf("empty script")
	}

	mapFS := make(fstest.MapFS)

	for _, script := range s.Modules {
		mapFS[script.Name] = &fstest.MapFile{
			Data: []byte(script.Content),
		}
	}

	mapFS[mainModule] = &fstest.MapFile{
		Data: []byte(s.Script.Content),
	}

	vm := sobek.New()

	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	vu := modules.NewModuleVU(ctx, vm)
	r := modules.NewResolver(mapFS, goModules, vu)

	m, err := r.Resolve(nil, mainModule)

	if err != nil {
		return err
	}

	p := m.(*sobek.SourceTextModuleRecord)

	if err = p.Link(); err != nil {
		return fmt.Errorf("failed to link source text: %w", err)
	}

	promise := vm.CyclicModuleRecordEvaluate(p, r.Resolve)

	if promise.State() != sobek.PromiseStateFulfilled {
		err = promise.Result().Export().(error)
		return fmt.Errorf("failed to evaluate promise: %w", err)
	}

	// main module ask for handler
	val := vm.GetModuleInstance(m).GetBindingValue(defaultExport)

	fn, ok := sobek.AssertFunction(val)
	if !ok {
		return fmt.Errorf("failed to assert default export function")
	}

	h.handler = fn
	h.runtime = vm

	return nil
}

func (h *Component) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:          RequestPort,
			Label:         "Request",
			Position:      module.Left,
			Configuration: Request{},
		},
		{
			Name:          ResponsePort,
			Position:      module.Right,
			Label:         "Response",
			Source:        true,
			Configuration: Response{},
		},
		{
			Name:          v1alpha1.SettingsPort,
			Label:         "Settings",
			Configuration: h.settings,
		},
	}
	if !h.settings.EnableErrorPort {
		return ports
	}
	return append(ports, module.Port{
		Position:      module.Bottom,
		Name:          ErrorPort,
		Label:         "Error",
		Source:        true,
		Configuration: Error{},
	})
}

func (h *Component) Instance() module.Component {
	return &Component{
		settings: defaultEngineSettings,
	}
}

var _ module.Component = (*Component)(nil)

func init() {
	registry.Register(&Component{})
}
