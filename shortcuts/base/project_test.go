// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/httpmock"
)

// ---------------------------------------------------------------------------
// User Story 1: Create Project
// ---------------------------------------------------------------------------

func TestProjectCreateExecute(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Step 1: Create base
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"app_token": "app_proj1", "name": "My Project"},
		},
	})

	// Step 2: Create _project_info table
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "tbl_info", "name": "_project_info"},
		},
	})

	// Step 3: Create fields for _project_info table (name, description, status)
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_info/fields",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "fld_name", "name": "name", "type": "text"},
		},
	})
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_info/fields",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "fld_desc", "name": "description", "type": "text"},
		},
	})
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_info/fields",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "fld_status", "name": "status", "type": "text"},
		},
	})

	// Step 4: Create initial project info record
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_info/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"record_id": "rec_info1"},
		},
	})

	args := []string{"+project-create", "--name", "My Project", "--folder-token", "fld_root"}
	if err := runShortcut(t, ProjectCreate, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"created": true`) {
		t.Fatalf("missing created flag, stdout=%s", got)
	}
	if !strings.Contains(got, `"app_proj1"`) {
		t.Fatalf("missing project_id, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// User Story 2: Get Project Overview
// ---------------------------------------------------------------------------

func TestProjectGetExecute(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Base info
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"app_token": "app_proj1", "name": "My Project"},
		},
	})

	// Table list (mix of well-known and custom tables)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_info", "name": "_project_info"},
					map[string]interface{}{"id": "tbl_members", "name": "_members"},
					map[string]interface{}{"id": "tbl_kv", "name": "_kv"},
					map[string]interface{}{"id": "tbl_tasks", "name": "_tasks"},
					map[string]interface{}{"id": "tbl_custom1", "name": "Requirements"},
				},
				"total": 5,
			},
		},
	})

	// Records for _project_info table (compact format)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_info/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"My Project", "A great project", "active"}},
				"fields":         []interface{}{"name", "description", "status"},
				"record_id_list": []interface{}{"rec_info1"},
				"has_more":       false,
			},
		},
	})

	// Records for _members table (compact format)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_members/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data": []interface{}{
					[]interface{}{"Alice", "owner"},
					[]interface{}{"Bob", "member"},
					[]interface{}{"Charlie", "member"},
				},
				"fields":         []interface{}{"name", "role"},
				"record_id_list": []interface{}{"rec_m1", "rec_m2", "rec_m3"},
				"has_more":       false,
				"total":          3,
			},
		},
	})

	// Records for _kv table (compact format)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"version", "1.0"}},
				"fields":         []interface{}{"key", "value"},
				"record_id_list": []interface{}{"rec_kv1"},
				"has_more":       false,
				"total":          5,
			},
		},
	})

	// Records for _tasks table (compact format)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_tasks/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"Fix bug", "pending"}},
				"fields":         []interface{}{"title", "status"},
				"record_id_list": []interface{}{"rec_t1"},
				"has_more":       false,
				"total":          10,
			},
		},
	})

	args := []string{"+project-get", "--base-token", "app_proj1"}
	if err := runShortcut(t, ProjectGet, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"members"`) {
		t.Fatalf("missing members section, stdout=%s", got)
	}
	if !strings.Contains(got, `"kv_count"`) || !strings.Contains(got, "5") {
		t.Fatalf("missing kv_count, stdout=%s", got)
	}
	if !strings.Contains(got, `"tasks"`) {
		t.Fatalf("missing tasks stats, stdout=%s", got)
	}
	if !strings.Contains(got, `"custom_tables"`) || !strings.Contains(got, `"Requirements"`) {
		t.Fatalf("missing custom_tables, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// User Story 3: Set KV (new key + update existing key)
// ---------------------------------------------------------------------------

func TestProjectKVSetExecuteNewKey(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list (KV table exists)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_kv", "name": "_kv"},
				},
				"total": 1,
			},
		},
	})

	// List records to search for existing key (not found)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{},
				"fields":         []interface{}{"key", "value"},
				"record_id_list": []interface{}{},
				"has_more":       false,
			},
		},
	})

	// Create new record
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"record_id": "rec_kv_new"},
		},
	})

	args := []string{"+project-kv-set", "--base-token", "app_proj1", "--key", "version", "--value", "2.0"}
	if err := runShortcut(t, ProjectKVSet, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"created": true`) {
		t.Fatalf("missing created flag, stdout=%s", got)
	}
}

func TestProjectKVSetExecuteUpdateExistingKey(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_kv", "name": "_kv"},
				},
				"total": 1,
			},
		},
	})

	// List records to search for existing key (found)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"version", "1.0"}},
				"fields":         []interface{}{"key", "value"},
				"record_id_list": []interface{}{"rec_kv_exist"},
				"has_more":       false,
			},
		},
	})

	// Update existing record
	reg.Register(&httpmock.Stub{
		Method: "PATCH",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records/rec_kv_exist",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"record_id": "rec_kv_exist"},
		},
	})

	args := []string{"+project-kv-set", "--base-token", "app_proj1", "--key", "version", "--value", "2.0"}
	if err := runShortcut(t, ProjectKVSet, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"updated": true`) {
		t.Fatalf("missing updated flag, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// User Story 4: Get KV
// ---------------------------------------------------------------------------

func TestProjectKVGetExecuteFound(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_kv", "name": "_kv"},
				},
				"total": 1,
			},
		},
	})

	// List records to search for key (found)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"version", "2.0"}},
				"fields":         []interface{}{"key", "value"},
				"record_id_list": []interface{}{"rec_kv1"},
				"has_more":       false,
			},
		},
	})

	args := []string{"+project-kv-get", "--base-token", "app_proj1", "--key", "version"}
	if err := runShortcut(t, ProjectKVGet, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"key"`) || !strings.Contains(got, `"version"`) {
		t.Fatalf("missing key in output, stdout=%s", got)
	}
	if !strings.Contains(got, `"value"`) || !strings.Contains(got, `"2.0"`) {
		t.Fatalf("missing value in output, stdout=%s", got)
	}
}

func TestProjectKVGetExecuteNotFound(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_kv", "name": "_kv"},
				},
				"total": 1,
			},
		},
	})

	// List records to search for key (not found)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{},
				"fields":         []interface{}{"key", "value"},
				"record_id_list": []interface{}{},
				"has_more":       false,
			},
		},
	})

	args := []string{"+project-kv-get", "--base-token", "app_proj1", "--key", "nonexistent"}
	err := runShortcut(t, ProjectKVGet, args, factory, stdout)
	// Either returns an error or empty result
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("unexpected error, err=%v", err)
		}
	} else {
		got := stdout.String()
		if strings.Contains(got, `"value"`) {
			t.Fatalf("should not contain value for nonexistent key, stdout=%s", got)
		}
	}
}

// ---------------------------------------------------------------------------
// User Story 5: List KV
// ---------------------------------------------------------------------------

func TestProjectKVListExecute(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_kv", "name": "_kv"},
				},
				"total": 1,
			},
		},
	})

	// List records (compact format)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data": []interface{}{
					[]interface{}{"version", "2.0"},
					[]interface{}{"env", "production"},
				},
				"fields":         []interface{}{"key", "value"},
				"record_id_list": []interface{}{"rec_kv1", "rec_kv2"},
				"has_more":       false,
			},
		},
	})

	args := []string{"+project-kv-list", "--base-token", "app_proj1"}
	if err := runShortcut(t, ProjectKVList, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"version"`) || !strings.Contains(got, `"2.0"`) {
		t.Fatalf("missing first kv pair, stdout=%s", got)
	}
	if !strings.Contains(got, `"env"`) || !strings.Contains(got, `"production"`) {
		t.Fatalf("missing second kv pair, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// User Story 6: Delete KV
// ---------------------------------------------------------------------------

func TestProjectKVDeleteExecute(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_kv", "name": "_kv"},
				},
				"total": 1,
			},
		},
	})

	// List records to search for key (found)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"old_key", "old_val"}},
				"fields":         []interface{}{"key", "value"},
				"record_id_list": []interface{}{"rec_kv_del"},
				"has_more":       false,
			},
		},
	})

	// Delete record
	reg.Register(&httpmock.Stub{
		Method: "DELETE",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_kv/records/rec_kv_del",
		Body:   map[string]interface{}{"code": 0, "data": map[string]interface{}{}},
	})

	args := []string{"+project-kv-delete", "--base-token", "app_proj1", "--key", "old_key"}
	if err := runShortcut(t, ProjectKVDelete, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"deleted": true`) {
		t.Fatalf("missing deleted flag, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// User Story 7: Add Member
// ---------------------------------------------------------------------------

func TestProjectMemberAddExecuteNewTable(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list (members table does not exist)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_info", "name": "_project_info"},
				},
				"total": 1,
			},
		},
	})

	// Create members table
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "tbl_members_new", "name": "_members"},
		},
	})

	// Create fields for members table (name, role, type, user_id, joined_at)
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_members_new/fields",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "fld_name", "name": "name", "type": "text"},
		},
	})
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_members_new/fields",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "fld_role", "name": "role", "type": "text"},
		},
	})
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_members_new/fields",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "fld_type", "name": "type", "type": "text"},
		},
	})
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_members_new/fields",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "fld_user_id", "name": "user_id", "type": "text"},
		},
	})
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_members_new/fields",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"id": "fld_joined_at", "name": "joined_at", "type": "text"},
		},
	})

	// Create member record
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_members_new/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"record_id": "rec_member1",
				"fields":    map[string]interface{}{"name": "Alice", "role": "owner"},
			},
		},
	})

	args := []string{"+project-member-add", "--base-token", "app_proj1", "--name", "Alice", "--role", "owner"}
	if err := runShortcut(t, ProjectMemberAdd, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"Alice"`) {
		t.Fatalf("missing member name, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// User Story 8: List Members
// ---------------------------------------------------------------------------

func TestProjectMemberListExecute(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_members", "name": "_members"},
				},
				"total": 1,
			},
		},
	})

	// List member records (compact format)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_members/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data": []interface{}{
					[]interface{}{"Alice", "owner"},
					[]interface{}{"Bob", "member"},
				},
				"fields":         []interface{}{"name", "role"},
				"record_id_list": []interface{}{"rec_m1", "rec_m2"},
				"has_more":       false,
			},
		},
	})

	args := []string{"+project-member-list", "--base-token", "app_proj1"}
	if err := runShortcut(t, ProjectMemberList, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"Alice"`) || !strings.Contains(got, `"Bob"`) {
		t.Fatalf("missing member names, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// User Story 9: Add Task
// ---------------------------------------------------------------------------

func TestProjectTaskAddExecute(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list (tasks table exists)
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_tasks", "name": "_tasks"},
				},
				"total": 1,
			},
		},
	})

	// Create task record
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_tasks/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"record_id": "rec_task_new",
				"fields": map[string]interface{}{
					"title":    "Implement login",
					"status":   "pending",
					"priority": 1,
				},
			},
		},
	})

	args := []string{"+project-task-add", "--base-token", "app_proj1", "--title", "Implement login", "--priority", "1"}
	if err := runShortcut(t, ProjectTaskAdd, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"Implement login"`) {
		t.Fatalf("missing task title, stdout=%s", got)
	}
	if !strings.Contains(got, `"pending"`) {
		t.Fatalf("missing pending status, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// User Story 10: Claim Next Task (task-next)
// ---------------------------------------------------------------------------

func TestProjectTaskNextExecute(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)

	// Table list
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"tables": []interface{}{
					map[string]interface{}{"id": "tbl_tasks", "name": "_tasks"},
				},
				"total": 1,
			},
		},
	})

	// List records to search for pending tasks
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_tasks/records",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data": []interface{}{
					[]interface{}{"Fix critical bug", "pending", "0", "1. Reproduce\n2. Debug\n3. Fix"},
				},
				"fields":         []interface{}{"title", "status", "priority", "subtasks"},
				"record_id_list": []interface{}{"rec_task_next"},
				"has_more":       false,
			},
		},
	})

	// Update task status to in_progress
	reg.Register(&httpmock.Stub{
		Method: "PATCH",
		URL:    "/open-apis/base/v3/bases/app_proj1/tables/tbl_tasks/records/rec_task_next",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"record_id": "rec_task_next",
				"fields": map[string]interface{}{
					"title":    "Fix critical bug",
					"status":   "in_progress",
					"priority": 0,
				},
			},
		},
	})

	args := []string{"+project-task-next", "--base-token", "app_proj1"}
	if err := runShortcut(t, ProjectTaskNext, args, factory, stdout); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, `"Fix critical bug"`) {
		t.Fatalf("missing task title, stdout=%s", got)
	}
	if !strings.Contains(got, `"subtasks"`) || !strings.Contains(got, "Reproduce") {
		t.Fatalf("missing subtasks, stdout=%s", got)
	}
}

// ---------------------------------------------------------------------------
// Dry-run tests
// ---------------------------------------------------------------------------

func TestDryRunProjectCreate(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(
		map[string]string{"name": "My Project", "folder-token": "fld_root"},
		nil, nil,
	)
	assertDryRunContains(t,
		ProjectCreate.DryRun(ctx, rt),
		"POST /open-apis/base/v3/bases",
	)
}

func TestDryRunProjectGet(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(
		map[string]string{"base-token": "app_proj1"},
		nil, nil,
	)
	assertDryRunContains(t,
		ProjectGet.DryRun(ctx, rt),
		"GET /open-apis/base/v3/bases/app_proj1",
		"/tables",
	)
}

func TestDryRunProjectKVSet(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(
		map[string]string{"base-token": "app_proj1", "key": "version", "value": "2.0"},
		nil, nil,
	)
	assertDryRunContains(t,
		ProjectKVSet.DryRun(ctx, rt),
		"/open-apis/base/v3/bases/app_proj1/tables",
		"records",
	)
}

func TestDryRunProjectKVGet(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(
		map[string]string{"base-token": "app_proj1", "key": "version"},
		nil, nil,
	)
	assertDryRunContains(t,
		ProjectKVGet.DryRun(ctx, rt),
		"/open-apis/base/v3/bases/app_proj1/tables",
		"records/search",
	)
}

func TestDryRunProjectTaskAdd(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(
		map[string]string{"base-token": "app_proj1", "title": "New task"},
		nil,
		map[string]int{"priority": 1},
	)
	assertDryRunContains(t,
		ProjectTaskAdd.DryRun(ctx, rt),
		"/open-apis/base/v3/bases/app_proj1/tables",
		"records",
	)
}

func TestDryRunProjectTaskNext(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(
		map[string]string{"base-token": "app_proj1"},
		nil, nil,
	)
	assertDryRunContains(t,
		ProjectTaskNext.DryRun(ctx, rt),
		"/open-apis/base/v3/bases/app_proj1/tables",
		"records/search",
	)
}

func TestDryRunProjectMemberAdd(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(
		map[string]string{"base-token": "app_proj1", "name": "Alice", "role": "owner"},
		nil, nil,
	)
	assertDryRunContains(t,
		ProjectMemberAdd.DryRun(ctx, rt),
		"/open-apis/base/v3/bases/app_proj1/tables",
		"records",
	)
}
