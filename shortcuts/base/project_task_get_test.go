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

func TestProjectTaskGetWithAttachments(t *testing.T) {
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
		URL:    "/open-apis/base/v3/bases/app_tget/tables",
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

	// Task record with attachments (standard format).
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_tget/tables/tbl_tasks/records/rec_with_attach",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"fields": map[string]interface{}{
					"title":   "生成配图",
					"status":  "done",
					"summary": "4张图生成完成",
					"result":  `[{"name":"scene1.png","time":"0-5s"},{"name":"scene2.png","time":"5-15s"}]`,
					"attachments": []interface{}{
						map[string]interface{}{
							"file_token": "boxcn001",
							"name":       "scene1.png",
							"mime_type":  "image/png",
							"size":       12345,
							"url":        "https://example.com/scene1.png",
						},
						map[string]interface{}{
							"file_token": "boxcn002",
							"name":       "scene2.png",
							"mime_type":  "image/png",
							"size":       12346,
							"url":        "https://example.com/scene2.png",
						},
					},
				},
				"record_id": "rec_with_attach",
			},
		},
	})

	shortcut := ProjectTaskGet
	shortcut.AuthTypes = []string{"bot"}
	parent := &cobra.Command{Use: "base"}
	shortcut.Mount(parent, factory)
	parent.SetArgs([]string{
		"+project-task-get",
		"--base-token", "app_tget",
		"--task-id", "rec_with_attach",
	})
	parent.SilenceErrors = true
	parent.SilenceUsage = true
	stdout.Reset()

	if err := parent.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("err=%v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "生成配图") {
		t.Fatalf("expected title, got: %s", out)
	}
	if !strings.Contains(out, "boxcn001") || !strings.Contains(out, "boxcn002") {
		t.Fatalf("expected both file tokens, got: %s", out)
	}
	if !strings.Contains(out, "scene1.png") || !strings.Contains(out, "scene2.png") {
		t.Fatalf("expected filenames, got: %s", out)
	}
	if !strings.Contains(out, "4张图生成完成") {
		t.Fatalf("expected summary, got: %s", out)
	}
	// task_result contains the JSON description.
	if !strings.Contains(out, "0-5s") {
		t.Fatalf("expected result JSON content, got: %s", out)
	}
}

func TestProjectTaskGetCompactFormat(t *testing.T) {
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
		URL:    "/open-apis/base/v3/bases/app_tget2/tables",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"items": []interface{}{map[string]interface{}{"table_id": "tbl_tasks", "name": "_tasks"}},
				"total": 1,
			},
		},
	})

	// Compact format (no attachments).
	reg.Register(&httpmock.Stub{
		Method: "GET",
		URL:    "/open-apis/base/v3/bases/app_tget2/tables/tbl_tasks/records/rec_compact",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"data":           []interface{}{[]interface{}{"写脚本", "done", "4个分镜脚本完成", "场景1..."}},
				"fields":         []interface{}{"title", "status", "summary", "result"},
				"record_id_list": []interface{}{"rec_compact"},
			},
		},
	})

	shortcut := ProjectTaskGet
	shortcut.AuthTypes = []string{"bot"}
	parent := &cobra.Command{Use: "base"}
	shortcut.Mount(parent, factory)
	parent.SetArgs([]string{
		"+project-task-get",
		"--base-token", "app_tget2",
		"--task-id", "rec_compact",
	})
	parent.SilenceErrors = true
	parent.SilenceUsage = true
	stdout.Reset()

	if err := parent.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("err=%v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "写脚本") {
		t.Fatalf("expected title, got: %s", out)
	}
	if !strings.Contains(out, "4个分镜脚本完成") {
		t.Fatalf("expected summary, got: %s", out)
	}
	if !strings.Contains(out, "场景1") {
		t.Fatalf("expected result content, got: %s", out)
	}
}
