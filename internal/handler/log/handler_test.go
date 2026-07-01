package log

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"

	"github.com/geoberle/pulse/internal/workitem"
)

func TestLogHandler(t *testing.T) {
	pr := workitem.MakeTestItem(workitem.KindPR, "pr:org/repo:1")
	prUpdated := workitem.MakeTestItem(workitem.KindPR, "pr:org/repo:1")
	prUpdated.Status.Phase = "merged"
	check := workitem.MakeTestItem(workitem.KindCheck, "check:100")

	tests := []struct {
		name       string
		call       func(h interface{})
		wantSubstr []string
	}{
		{
			name: "add logs item",
			call: func(h interface{}) {
				h.(interface{ OnAdd(interface{}, bool) }).OnAdd(pr, false)
			},
			wantSubstr: []string{"Added", "pr:org/repo:1", "Test PR", "open"},
		},
		{
			name: "delete logs item",
			call: func(h interface{}) {
				h.(interface{ OnDelete(interface{}) }).OnDelete(pr)
			},
			wantSubstr: []string{"Deleted", "pr:org/repo:1"},
		},
		{
			name: "update logs new item",
			call: func(h interface{}) {
				h.(interface {
					OnUpdate(interface{}, interface{})
				}).OnUpdate(pr, prUpdated)
			},
			wantSubstr: []string{"Updated", "pr:org/repo:1", "merged"},
		},
		{
			name: "add check logs kind",
			call: func(h interface{}) {
				h.(interface{ OnAdd(interface{}, bool) }).OnAdd(check, false)
			},
			wantSubstr: []string{"Added", "check:100", "ci"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := funcr.New(func(prefix, args string) {
				buf.WriteString(prefix + " " + args)
			}, funcr.Options{})

			h := NewHandler(log)
			tt.call(h)

			got := buf.String()
			for _, want := range tt.wantSubstr {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\ngot: %s", want, got)
				}
			}
		})
	}
}
