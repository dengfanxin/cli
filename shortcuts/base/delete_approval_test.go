// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/httpmock"
)

func registerDeleteAuthCodeValidationStub(reg *httpmock.Registry, baseToken string) *httpmock.Stub {
	stub := &httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/" + baseToken + "/delete_auth_codes/validate",
		Body:   map[string]interface{}{"code": 0, "data": map[string]interface{}{"request_id": "delete_req_1"}},
	}
	reg.Register(stub)
	return stub
}

func TestDeleteApprovalPrepareDoesNotCallRealDelete(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)
	reg.Register(&httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/base/v3/bases/app_x/delete_approval_requests",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{"request_id": "delete_req_1", "approval_url": "https://example.test/request/delete_req_1"},
		},
	})

	err := runShortcut(t, BaseTableDelete, []string{
		"+table-delete", "--base-token", "app_x", "--table-id", "tbl_x", "--prepare-approval", "--yes",
	}, factory, stdout)
	if err != nil {
		t.Fatalf("prepare approval err=%v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "delete_req_1") || !strings.Contains(got, "approval_url") {
		t.Fatalf("stdout=%s", got)
	}
}

func TestDeleteApprovalRequiresAuthCode(t *testing.T) {
	factory, stdout, _ := newExecuteFactory(t)

	err := runShortcut(t, BaseTableDelete, []string{
		"+table-delete", "--base-token", "app_x", "--table-id", "tbl_x", "--yes",
	}, factory, stdout)
	if err == nil || !strings.Contains(err.Error(), "--auth-code is required") {
		t.Fatalf("err=%v", err)
	}
}

func TestDeleteApprovalValidatesAuthCodeBeforeSingleRealDelete(t *testing.T) {
	factory, stdout, reg := newExecuteFactory(t)
	registerTokenStub(reg)
	validateStub := registerDeleteAuthCodeValidationStub(reg, "app_x")
	deleteStub := &httpmock.Stub{
		Method: "DELETE",
		URL:    "/open-apis/base/v3/bases/app_x/tables/tbl_x",
		Body:   map[string]interface{}{"code": 0, "data": map[string]interface{}{}},
	}
	reg.Register(deleteStub)

	err := runShortcut(t, BaseTableDelete, []string{
		"+table-delete", "--base-token", "app_x", "--table-id", "tbl_x", "--auth-code", "larkauth_v1_test", "--yes",
	}, factory, stdout)
	if err != nil {
		t.Fatalf("delete err=%v", err)
	}
	var validateBody map[string]interface{}
	if err := json.Unmarshal(validateStub.CapturedBody, &validateBody); err != nil {
		t.Fatalf("validate body=%q err=%v", validateStub.CapturedBody, err)
	}
	if got := validateBody["auth_code"]; got != "larkauth_v1_test" {
		t.Fatalf("validate auth_code=%v", got)
	}
	if got := validateBody["action"]; got != "base.table.delete" {
		t.Fatalf("validate action=%v", got)
	}
	if got := deleteStub.CapturedHeaders.Get("X-Lark-Delete-Auth-Code"); got != "" {
		t.Fatalf("original delete protocol changed with auth header=%q", got)
	}
}
