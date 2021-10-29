package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_detectOperations(t *testing.T) {
	tests := []struct {
		name string
		apps []v1alpha1.Application
		want map[string][]operation
	}{
		{
			name: "ok",
			apps: []v1alpha1.Application{
				v1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-app",
						Annotations: map[string]string{
							"argoswitch.github.io/primary":     "sync",
							"argoswitch.github.io/secondary":   "disable",
							"argoswitch.github.io/service-out": "delete",
							"argoswitch.github.io/maint":       "disable",
						},
					},
				},
			},
			want: map[string][]operation{
				"primary": []operation{
					operation{
						Name:      "test-app",
						Operation: "sync",
					},
				},
				"secondary": []operation{
					operation{
						Name:      "test-app",
						Operation: "disable",
					},
				},
				"service-out": []operation{
					operation{
						Name:      "test-app",
						Operation: "delete",
					},
				},
				"maint": []operation{
					operation{
						Name:      "test-app",
						Operation: "disable",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectOperations(tt.apps); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("detectOperations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_render(t *testing.T) {
	type args struct {
		affects      map[string][]operation
		rs           []operation
		currentState string
	}
	tests := []struct {
		name    string
		args    args
		wantW   []string
		wantErr bool
	}{
		{

			name: "ok",
			args: args{
				affects: map[string][]operation{
					"primary": []operation{
						operation{
							Name:      "test-app1",
							Operation: "sync",
							Error:     nil,
						},
					},
				},
				rs: []operation{
					operation{
						Name:      "test-app2",
						Operation: "delete",
						Error:     errors.New("test error"),
					},
					operation{
						Name:      "test-app3",
						Operation: "sync",
						Error:     nil,
					},
				},
				currentState: "primary",
			},

			wantW: []string{
				"test-app1 will be sync",
				"test-app2 has error: test error",
				"test-app3 is synced",
				"btn btn-primary disabled",
				`<span class="fs-4 text-decoration-underline">primary</span>`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := render(w, tt.args.affects, tt.args.rs, tt.args.currentState); (err != nil) != tt.wantErr {
				t.Errorf("render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, want := range tt.wantW {
				if gotW := w.String(); !strings.Contains(gotW, want) {
					t.Errorf("render() = %v, want %v", gotW, want)
				}
			}
		})
	}
}
