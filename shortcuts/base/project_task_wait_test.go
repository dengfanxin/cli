// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/spf13/cobra"
)

func TestProjectTaskWaitAlreadyDone(t *testing.T) {
	config := &core.CliConfig{
		AppID:      "test-app-" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "-"),
		AppSecret:  "test-secret",
		Brand:      core.BrandFeishu,
		UserOpenId: "ou_testuser",
	}
	factory, stdout, _, reg := cmdutil.TestFactory(t, config)

	registerTokenStub(reg)

	// Table list.
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_wait/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"table_id": "tbl_tasks", "name": "_tasks"},
				},
				"total": 1,
			},
		},
	})

	// Task record: already done.
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_wait/tables/tbl_tasks/records/rec_done",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"done", "脚本写完了", "写脚本"}},
				"fields":         []interface{}{"status", "summary", "title"},
				"record_id_list": []interface{}{"rec_done"},
				"has_more":       false,
			},
		},
	})

	shortcut := ProjectTaskWait
	shortcut.AuthTypes = []string{"bot"}
	parent := &cobra.Command{Use: "base"}
	shortcut.Mount(parent, factory)
	parent.SetArgs([]string{
		"+project-task-wait",
		"--base-token", "app_wait",
		"--task-id", "rec_done",
	})
	parent.SilenceErrors = true
	parent.SilenceUsage = true
	stdout.Reset()

	if err := parent.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("err=%v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, `"reached": true`) && !strings.Contains(out, `"reached":true`) {
		t.Fatalf("expected reached:true, got: %s", out)
	}
	if !strings.Contains(out, "写脚本") {
		t.Fatalf("expected task title, got: %s", out)
	}
	if !strings.Contains(out, "脚本写完了") {
		t.Fatalf("expected summary, got: %s", out)
	}
}

func TestProjectTaskWaitBlocked(t *testing.T) {
	config := &core.CliConfig{
		AppID:      "test-app-" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "-"),
		AppSecret:  "test-secret",
		Brand:      core.BrandFeishu,
		UserOpenId: "ou_testuser",
	}
	factory, stdout, _, reg := cmdutil.TestFactory(t, config)

	registerTokenStub(reg)

	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_wait2/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"table_id": "tbl_tasks", "name": "_tasks"},
				},
				"total": 1,
			},
		},
	})

	// Task is blocked — wait for "blocked" status.
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_wait2/tables/tbl_tasks/records/rec_blocked",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"blocked", "缺少权限", "数据迁移"}},
				"fields":         []interface{}{"status", "summary", "title"},
				"record_id_list": []interface{}{"rec_blocked"},
				"has_more":       false,
			},
		},
	})

	shortcut := ProjectTaskWait
	shortcut.AuthTypes = []string{"bot"}
	parent := &cobra.Command{Use: "base"}
	shortcut.Mount(parent, factory)
	parent.SetArgs([]string{
		"+project-task-wait",
		"--base-token", "app_wait2",
		"--task-id", "rec_blocked",
		"--status", "blocked",
	})
	parent.SilenceErrors = true
	parent.SilenceUsage = true
	stdout.Reset()

	if err := parent.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("err=%v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, `"reached"`) {
		t.Fatalf("expected reached, got: %s", out)
	}
	if !strings.Contains(out, "blocked") {
		t.Fatalf("expected blocked status, got: %s", out)
	}
}
