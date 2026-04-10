// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/shortcuts/common"
)

func baseDownloadTestConfig() *core.CliConfig {
	return &core.CliConfig{
		AppID: "base-download-test-app", AppSecret: "test-secret", Brand: core.BrandFeishu,
	}
}

func mountAndRunBase(t *testing.T, s common.Shortcut, args []string, f *cmdutil.Factory, stdout *bytes.Buffer) error {
	t.Helper()
	parent := &cobra.Command{Use: "base"}
	s.Mount(parent, f)
	parent.SetArgs(args)
	parent.SilenceErrors = true
	parent.SilenceUsage = true
	if stdout != nil {
		stdout.Reset()
	}
	return parent.Execute()
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd error: %v", err)
		}
	})
}

// TestRecordDownloadAttachmentSuccess: download a Base attachment file successfully.
func TestRecordDownloadAttachmentSuccess(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, baseDownloadTestConfig())

	// Mock the medias download endpoint with raw bytes.
	reg.Register(&httpmock.Stub{
		Method:  "GET",
		URL:     "/open-apis/drive/v1/medias/file_token_123/download",
		Status:  200,
		Body:    []byte("PNG_FILE_CONTENT"),
		Headers: http.Header{"Content-Type": []string{"image/png"}},
	})

	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir)

	err := mountAndRunBase(t, BaseRecordDownloadAttachment, []string{
		"+record-download-attachment",
		"--base-token", "app_xxx",
		"--file-token", "file_token_123",
		"--output", "scene1.png",
		"--as", "bot",
	}, f, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile("scene1.png")
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != "PNG_FILE_CONTENT" {
		t.Fatalf("downloaded content = %q, want %q", string(data), "PNG_FILE_CONTENT")
	}

	out := stdout.String()
	if !strings.Contains(out, "scene1.png") {
		t.Fatalf("stdout missing saved path: %s", out)
	}
}

// TestRecordDownloadAttachmentRejectsOverwriteWithoutFlag: must not overwrite existing files.
func TestRecordDownloadAttachmentRejectsOverwriteWithoutFlag(t *testing.T) {
	f, _, _, _ := cmdutil.TestFactory(t, baseDownloadTestConfig())

	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir)

	if err := os.WriteFile("existing.png", []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	err := mountAndRunBase(t, BaseRecordDownloadAttachment, []string{
		"+record-download-attachment",
		"--base-token", "app_xxx",
		"--file-token", "file_token_123",
		"--output", "existing.png",
		"--as", "bot",
	}, f, nil)
	if err == nil {
		t.Fatal("expected overwrite protection error, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRecordDownloadAttachmentAllowsOverwrite: --overwrite replaces existing file.
func TestRecordDownloadAttachmentAllowsOverwrite(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, baseDownloadTestConfig())
	reg.Register(&httpmock.Stub{
		Method:  "GET",
		URL:     "/open-apis/drive/v1/medias/file_token_456/download",
		Status:  200,
		Body:    []byte("new content"),
		Headers: http.Header{"Content-Type": []string{"image/png"}},
	})

	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir)

	if err := os.WriteFile("existing.png", []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	err := mountAndRunBase(t, BaseRecordDownloadAttachment, []string{
		"+record-download-attachment",
		"--base-token", "app_xxx",
		"--file-token", "file_token_456",
		"--output", "existing.png",
		"--overwrite",
		"--as", "bot",
	}, f, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile("existing.png")
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != "new content" {
		t.Fatalf("downloaded content = %q, want %q", string(data), "new content")
	}
}

// TestRecordDownloadAttachmentDefaultOutput: --output defaults to file-token if not provided.
func TestRecordDownloadAttachmentDefaultOutput(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, baseDownloadTestConfig())

	reg.Register(&httpmock.Stub{
		Method:  "GET",
		URL:     "/open-apis/drive/v1/medias/file_token_default/download",
		Status:  200,
		Body:    []byte("default content"),
		Headers: http.Header{"Content-Type": []string{"image/png"}},
	})

	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir)

	err := mountAndRunBase(t, BaseRecordDownloadAttachment, []string{
		"+record-download-attachment",
		"--base-token", "app_xxx",
		"--file-token", "file_token_default",
		"--as", "bot",
	}, f, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat("file_token_default"); err != nil {
		t.Fatalf("expected file at default path, got: %v", err)
	}
}

// TestDryRunRecordDownloadAttachment: dry-run shows the API call with extra param.
func TestDryRunRecordDownloadAttachment(t *testing.T) {
	rt := newBaseTestRuntime(
		map[string]string{
			"base-token": "app_xxx",
			"file-token": "file_yyy",
			"output":     "out.png",
		},
		nil,
		nil,
	)
	dr := dryRunRecordDownloadAttachment(nil, rt)
	out := dr.Format()

	if !strings.Contains(out, "/open-apis/drive/v1/medias/") {
		t.Fatalf("dry-run missing medias endpoint: %s", out)
	}
	if !strings.Contains(out, "file_yyy") {
		t.Fatalf("dry-run missing file token: %s", out)
	}
	if !strings.Contains(out, "extra") {
		t.Fatalf("dry-run missing extra param: %s", out)
	}
	if !strings.Contains(out, "bitablePerm") {
		t.Fatalf("dry-run extra should reference bitablePerm: %s", out)
	}
}
