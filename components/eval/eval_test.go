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
