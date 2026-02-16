package eval

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/api/v1alpha1"
	"github.com/tiny-systems/module/module"
	"testing"
)

func TestComponent_Handle(t *testing.T) {
	type args struct {
		handler module.Handler
		port    string
		msg     interface{}
		wantErr bool
	}
	tests := []struct {
		name string
		args []args
	}{
		{
			name: "wrong settings, syntax error main script",

			args: []args{
				{
					port:    v1alpha1.SettingsPort,
					wantErr: true,
					msg: Settings{
						Script: Script{
							Content: `export default fction () {}`,
						},
					},
				},
			},
		},
		{
			name: "wrong settings, no default export function",

			args: []args{
				{
					port:    v1alpha1.SettingsPort,
					wantErr: true,
					msg: Settings{
						Script: Script{
							Content: `export named function () {}`,
						},
					},
				},
			},
		},
		{
			name: "success match result string",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default function () { return "result";}`,
						},
					},
				},
				{
					port: RequestPort,
					msg:  Request{},
					handler: func(_ context.Context, port string, data any) any {
						if port != ResponsePort {
							return fmt.Errorf("response sent to the wrong port: %s", port)
						}
						resp, ok := data.(Response)
						if !ok {
							return fmt.Errorf("response type is invalid")
						}
						if fmt.Sprint(resp.OutputData) != "result" {
							return fmt.Errorf("response sent the wrong result: %v", resp.OutputData)
						}
						return nil
					},
				},
			},
		},
		{
			name: "success match result check type int",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default function () { return 34;}`,
						},
					},
				},
				{
					port: RequestPort,
					msg:  Request{},
					handler: func(_ context.Context, port string, data any) any {
						if port != ResponsePort {
							return fmt.Errorf("response sent to the wrong port: %s", port)
						}
						resp, ok := data.(Response)
						if !ok {
							return fmt.Errorf("response type is invalid")
						}

						res, _ := resp.OutputData.(int64)
						if res != 34 {
							return fmt.Errorf("response sent the wrong result: %v", res)
						}
						return nil
					},
				},
			},
		},
		{
			name: "success match response use request",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default function (i) { return i + " world";}`,
						},
					},
				},
				{
					port: RequestPort,
					msg: Request{
						InputData: "hello",
					},
					handler: func(_ context.Context, port string, data any) any {
						if port != ResponsePort {
							return fmt.Errorf("response sent to the wrong port: %s", port)
						}
						resp, ok := data.(Response)
						if !ok {
							return fmt.Errorf("response type is invalid")
						}
						if fmt.Sprint(resp.OutputData) != "hello world" {
							return fmt.Errorf("response sent the wrong result: %v", resp.OutputData)
						}
						return nil
					},
				},
			},
		},

		{
			name: "success use promises",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default async function (i) { return await new Promise((resolve, reject) => {resolve("done")})}`,
						},
					},
				},
				{
					port: RequestPort,
					msg: Request{
						InputData: "hello",
					},
					handler: func(_ context.Context, port string, data any) any {
						if port != ResponsePort {
							return fmt.Errorf("response sent to the wrong port: %s", port)
						}
						resp, ok := data.(Response)
						if !ok {
							return fmt.Errorf("response type is invalid")
						}
						if fmt.Sprint(resp.OutputData) != "done" {
							return fmt.Errorf("response sent the wrong result: %v", resp.OutputData)
						}
						return nil
					},
				},
			},
		},
		{
			name: "reject should trigger error",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default async function (i) { return await new Promise((resolve, reject) => {reject("error")})}`,
						},
					},
				},
				{
					port: RequestPort,
					msg:  Request{},
					handler: func(_ context.Context, port string, data any) any {
						return nil
					},
					wantErr: true,
				},
			},
		},
		{
			name: "success promise of promise",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default async function (i) { return await new Promise((resolve, reject) => {resolve(new Promise((resolve, reject) => {resolve("done")}))})}`,
						},
					},
				},
				{
					port: RequestPort,
					msg: Request{
						InputData: "hello",
					},
					handler: func(_ context.Context, port string, data any) any {
						if port != ResponsePort {
							return fmt.Errorf("response sent to the wrong port: %s", port)
						}
						resp, ok := data.(Response)
						if !ok {
							return fmt.Errorf("response type is invalid")
						}
						if fmt.Sprint(resp.OutputData) != "done" {
							return fmt.Errorf("response sent the wrong result: %v", resp.OutputData)
						}
						return nil
					},
				},
			},
		},
		{
			name: "error port enabled, js throw routes to error port",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						EnableErrorPort: true,
						Script: Script{
							Content: `export default function () { throw new Error("something broke"); }`,
						},
					},
				},
				{
					port: RequestPort,
					msg: Request{
						Context: "trace-123",
					},
					handler: func(_ context.Context, port string, data any) any {
						if port != ErrorPort {
							return fmt.Errorf("expected error port, got: %s", port)
						}
						e, ok := data.(Error)
						if !ok {
							return fmt.Errorf("expected Error type")
						}
						if e.Context != "trace-123" {
							return fmt.Errorf("context not passed through: %v", e.Context)
						}
						if e.Error == "" {
							return fmt.Errorf("error message is empty")
						}
						return nil
					},
				},
			},
		},
		{
			name: "error port disabled, js throw returns error",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						EnableErrorPort: false,
						Script: Script{
							Content: `export default function () { throw new Error("fail"); }`,
						},
					},
				},
				{
					port:    RequestPort,
					msg:     Request{},
					wantErr: true,
				},
			},
		},
		{
			name: "context passthrough to response",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default function (i) { return i; }`,
						},
					},
				},
				{
					port: RequestPort,
					msg: Request{
						Context:   map[string]any{"traceId": "abc", "step": 42},
						InputData: "data",
					},
					handler: func(_ context.Context, port string, data any) any {
						resp := data.(Response)
						ctx, ok := resp.Context.(map[string]any)
						if !ok {
							return fmt.Errorf("context type lost: %T", resp.Context)
						}
						if ctx["traceId"] != "abc" {
							return fmt.Errorf("context traceId mismatch: %v", ctx["traceId"])
						}
						return nil
					},
				},
			},
		},
		{
			name: "context passthrough to error port",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						EnableErrorPort: true,
						Script: Script{
							Content: `export default function () { throw new Error("fail"); }`,
						},
					},
				},
				{
					port: RequestPort,
					msg: Request{
						Context: "my-ctx",
					},
					handler: func(_ context.Context, port string, data any) any {
						e := data.(Error)
						if e.Context != "my-ctx" {
							return fmt.Errorf("context not passed to error: %v", e.Context)
						}
						return nil
					},
				},
			},
		},
		{
			name: "local module import from includes",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `import {add} from "math.js"; export default function(i) { return add(i, 10); }`,
						},
						Modules: []ScriptItem{
							{
								Name:    "math.js",
								Content: `export function add(a, b) { return a + b; }`,
							},
						},
					},
				},
				{
					port: RequestPort,
					msg: Request{
						InputData: 5,
					},
					handler: func(_ context.Context, port string, data any) any {
						resp := data.(Response)
						res, _ := resp.OutputData.(int64)
						if res != 15 {
							return fmt.Errorf("expected 15, got: %v", resp.OutputData)
						}
						return nil
					},
				},
			},
		},
		{
			name: "object input and output",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default function(inp) { return {name: inp.name, upper: inp.name.toUpperCase()}; }`,
						},
					},
				},
				{
					port: RequestPort,
					msg: Request{
						InputData: map[string]any{"name": "alice"},
					},
					handler: func(_ context.Context, port string, data any) any {
						resp := data.(Response)
						out, ok := resp.OutputData.(map[string]any)
						if !ok {
							return fmt.Errorf("expected map output, got: %T", resp.OutputData)
						}
						if out["name"] != "alice" || out["upper"] != "ALICE" {
							return fmt.Errorf("unexpected output: %v", out)
						}
						return nil
					},
				},
			},
		},
		{
			name: "return undefined",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default function() {}`,
						},
					},
				},
				{
					port: RequestPort,
					msg:  Request{},
					handler: func(_ context.Context, port string, data any) any {
						resp := data.(Response)
						if resp.OutputData != nil {
							return fmt.Errorf("expected nil for undefined, got: %v", resp.OutputData)
						}
						return nil
					},
				},
			},
		},
		{
			name: "return null",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `export default function() { return null; }`,
						},
					},
				},
				{
					port: RequestPort,
					msg:  Request{},
					handler: func(_ context.Context, port string, data any) any {
						resp := data.(Response)
						if resp.OutputData != nil {
							return fmt.Errorf("expected nil for null, got: %v", resp.OutputData)
						}
						return nil
					},
				},
			},
		},
		{
			name: "empty script content",
			args: []args{
				{
					port:    v1alpha1.SettingsPort,
					wantErr: true,
					msg: Settings{
						Script: Script{
							Content: "",
						},
					},
				},
			},
		},
		{
			name: "request before settings returns error",
			args: []args{
				{
					port:    RequestPort,
					msg:     Request{},
					wantErr: true,
				},
			},
		},
		{
			name: "multiple requests reuse compiled script",
			args: []args{
				{
					port: v1alpha1.SettingsPort,
					msg: Settings{
						Script: Script{
							Content: `let counter = 0; export default function() { counter++; return counter; }`,
						},
					},
				},
				{
					port: RequestPort,
					msg:  Request{},
					handler: func(_ context.Context, port string, data any) any {
						resp := data.(Response)
						res, _ := resp.OutputData.(int64)
						if res != 1 {
							return fmt.Errorf("first call expected 1, got: %v", resp.OutputData)
						}
						return nil
					},
				},
				{
					port: RequestPort,
					msg:  Request{},
					handler: func(_ context.Context, port string, data any) any {
						resp := data.(Response)
						res, _ := resp.OutputData.(int64)
						if res != 2 {
							return fmt.Errorf("second call expected 2, got: %v", resp.OutputData)
						}
						return nil
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := (&Component{}).Instance()
			for _, a := range tt.args {
				if err := h.Handle(context.Background(), a.handler, a.port, a.msg); (err != nil) != a.wantErr {
					t.Errorf("Handle() error = %v, wantErr %v", err, a.wantErr)
				}
			}
		})
	}
}
