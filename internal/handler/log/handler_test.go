package log

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"

	"github.com/geoberle/pulse/internal/informer"
	"github.com/geoberle/pulse/internal/workitem"
)

func TestOnEvent(t *testing.T) {
	pr := workitem.MakeTestItem(workitem.KindPR, "pr:org/repo:1", "fix bug")
	prUpdated := workitem.MakeTestItem(workitem.KindPR, "pr:org/repo:1", "fix bug")
	prUpdated.Status = "merged"
	check := workitem.MakeTestItem(workitem.KindCheck, "check:100", "lint")

	tests := []struct {
		name       string
		event      informer.Event
		wantSubstr []string
	}{
		{
			name: "added event logs new item",
			event: informer.Event{
				Type: informer.EventAdded,
				New:  pr,
			},
			wantSubstr: []string{"Added", "pr:org/repo:1", "fix bug", "open"},
		},
		{
			name: "deleted event logs old item",
			event: informer.Event{
				Type: informer.EventDeleted,
				Old:  pr,
			},
			wantSubstr: []string{"Deleted", "pr:org/repo:1"},
		},
		{
			name: "updated event logs new item",
			event: informer.Event{
				Type: informer.EventUpdated,
				Old:  pr,
				New:  prUpdated,
			},
			wantSubstr: []string{"Updated", "pr:org/repo:1", "merged"},
		},
		{
			name: "event with parent includes parent id",
			event: informer.Event{
				Type:   informer.EventAdded,
				New:    check,
				Parent: pr,
			},
			wantSubstr: []string{"Added", "check:100", "parent", "pr:org/repo:1"},
		},
		{
			name: "event without parent omits parent field",
			event: informer.Event{
				Type: informer.EventAdded,
				New:  check,
			},
			wantSubstr: []string{"Added", "check:100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := funcr.New(func(prefix, args string) {
				buf.WriteString(prefix + " " + args)
			}, funcr.Options{})

			h := NewHandler(log)
			h.OnEvent(tt.event)

			got := buf.String()
			for _, want := range tt.wantSubstr {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\ngot: %s", want, got)
				}
			}
		})
	}
}
