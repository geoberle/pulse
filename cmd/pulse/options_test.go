package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jsonstore "github.com/geoberle/pulse/internal/store/json"
	"github.com/geoberle/pulse/internal/workitem"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	validConfig := "jira:\n  host: https://example.atlassian.net\n  email: test@example.com\n  token: test-token\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\n"
	validPrompts := "review_comment: test\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n"

	tests := []struct {
		name    string
		config  string
		prompts string
		wantErr bool
	}{
		{
			name:    "missing config file",
			config:  "",
			prompts: "",
			wantErr: true,
		},
		{
			name:    "no repos",
			config:  "jira:\n  host: https://example.atlassian.net\njira_project: ARO\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "missing jira_project",
			config:  "jira:\n  host: https://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "missing jira host",
			config:  "repos:\n  - Azure/ARO-HCP\njira_project: ARO\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "http jira host",
			config:  "jira:\n  host: http://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "invalid poll_interval",
			config:  "jira:\n  host: https://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\npoll_interval: notaduration\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "invalid prompt template",
			config:  validConfig,
			prompts: "review_comment: \"{{.Broken\"\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if len(tt.config) == 0 {
				opts := &RawOptions{
					ConfigFile:  "/nonexistent/config.yaml",
					PromptsFile: "/nonexistent/prompts.yaml",
				}
				_, err := opts.Validate()
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			tmp := t.TempDir()
			cfgPath := filepath.Join(tmp, "config.yaml")
			promptsPath := filepath.Join(tmp, "prompts.yaml")

			if err := os.WriteFile(cfgPath, []byte(tt.config), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(promptsPath, []byte(tt.prompts), 0644); err != nil {
				t.Fatal(err)
			}

			opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
			_, err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type fakeSource struct {
	items []*workitem.WorkItem
	err   error
	calls int
}

func (f *fakeSource) List(_ context.Context) ([]*workitem.WorkItem, error) {
	f.calls++
	return f.items, f.err
}

func testItem(name string) *workitem.WorkItem {
	return &workitem.WorkItem{
		TypeMeta:   metav1.TypeMeta{Kind: string(workitem.KindJira), APIVersion: workitem.APIVersion},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Test"}`),
		Status:     workitem.WorkItemStatus{Phase: "New"},
	}
}

func TestCachedSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		storeItems    []*workitem.WorkItem
		storeErr      error
		liveItems     []*workitem.WorkItem
		wantItems     []*workitem.WorkItem
		wantLiveCalls int
	}{
		{
			name:          "returns loaded items on first call",
			storeItems:    []*workitem.WorkItem{testItem("jira.aro-1")},
			liveItems:     []*workitem.WorkItem{testItem("jira.aro-2")},
			wantItems:     []*workitem.WorkItem{testItem("jira.aro-1")},
			wantLiveCalls: 0,
		},
		{
			name:          "falls through to live on empty load",
			storeItems:    nil,
			liveItems:     []*workitem.WorkItem{testItem("jira.aro-2")},
			wantItems:     []*workitem.WorkItem{testItem("jira.aro-2")},
			wantLiveCalls: 1,
		},
		{
			name:          "falls through to live on load error",
			storeErr:      fmt.Errorf("disk on fire"),
			liveItems:     []*workitem.WorkItem{testItem("jira.aro-2")},
			wantItems:     []*workitem.WorkItem{testItem("jira.aro-2")},
			wantLiveCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmp := t.TempDir()
			storePath := filepath.Join(tmp, "state.json")

			store, err := jsonstore.New(storePath, logr.Discard())
			if err != nil {
				t.Fatal(err)
			}

			if tt.storeErr != nil {
				// Write invalid JSON to trigger load error
				if err := os.WriteFile(storePath, []byte("{invalid"), 0644); err != nil {
					t.Fatal(err)
				}
			} else if tt.storeItems != nil {
				if err := store.Save(tt.storeItems); err != nil {
					t.Fatal(err)
				}
			}

			live := &fakeSource{items: tt.liveItems}
			src := &cachedSource{
				store: store,
				live:  live,
				log:   logr.Discard(),
				liveC: make(chan struct{}),
			}

			got, err := src.List(context.Background())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(got) != len(tt.wantItems) {
				t.Fatalf("got %d items, want %d", len(got), len(tt.wantItems))
			}
			for i, item := range got {
				if item.Name != tt.wantItems[i].Name {
					t.Errorf("item[%d].Name = %s, want %s", i, item.Name, tt.wantItems[i].Name)
				}
			}
			if live.calls != tt.wantLiveCalls {
				t.Errorf("live source called %d times, want %d", live.calls, tt.wantLiveCalls)
			}
		})
	}
}

func TestCachedSource_SecondCallDelegatesToLive(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	storePath := filepath.Join(tmp, "state.json")

	store, err := jsonstore.New(storePath, logr.Discard())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save([]*workitem.WorkItem{testItem("jira.aro-1")}); err != nil {
		t.Fatal(err)
	}

	live := &fakeSource{items: []*workitem.WorkItem{testItem("jira.aro-2")}}
	src := &cachedSource{
		store: store,
		live:  live,
		log:   logr.Discard(),
		liveC: make(chan struct{}),
	}

	// First call: returns store items
	got, err := src.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "jira.aro-1" {
		t.Errorf("first call: got %s, want jira.aro-1", got[0].Name)
	}
	if live.calls != 0 {
		t.Errorf("first call: live called %d times, want 0", live.calls)
	}

	// Second call: delegates to live
	got, err = src.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "jira.aro-2" {
		t.Errorf("second call: got %s, want jira:ARO-2", got[0].Name)
	}
	if live.calls != 1 {
		t.Errorf("second call: live called %d times, want 1", live.calls)
	}
}

func TestCachedSource_CorruptFileFallsThroughToLive(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	storePath := filepath.Join(tmp, "state.json")

	store, err := jsonstore.New(storePath, logr.Discard())
	if err != nil {
		t.Fatal(err)
	}

	// Write invalid JSON — Store.Load tolerates this (returns nil, nil)
	if err := os.WriteFile(storePath, []byte("{invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	live := &fakeSource{items: []*workitem.WorkItem{testItem("jira.aro-1")}}
	src := &cachedSource{
		store: store,
		live:  live,
		log:   logr.Discard(),
		liveC: make(chan struct{}),
	}

	// First call: corrupt file tolerated, falls through to live
	got, err := src.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "jira.aro-1" {
		t.Errorf("got %s, want jira:ARO-1", got[0].Name)
	}
	if live.calls != 1 {
		t.Errorf("live called %d times, want 1", live.calls)
	}

	// Second call: seeded=true, goes straight to live
	got, err = src.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "jira.aro-1" {
		t.Errorf("got %s, want jira:ARO-1", got[0].Name)
	}
	if live.calls != 2 {
		t.Errorf("live called %d times, want 2", live.calls)
	}
}

func TestValidate_Success(t *testing.T) {
	t.Parallel()
	opts := &RawOptions{
		ConfigFile:  filepath.Join("..", "..", "config", "default_config.yaml"),
		PromptsFile: filepath.Join("..", "..", "config", "default_prompts.yaml"),
	}
	validated, err := opts.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if validated.Config.JiraProject != "ARO" {
		t.Errorf("expected jira_project ARO, got %s", validated.Config.JiraProject)
	}
}
