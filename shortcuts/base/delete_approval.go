// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

const (
	deleteApprovalRequestPath  = "/open-apis/base/v3/bases/%s/delete_approval_requests"
	deleteAuthCodeValidatePath = "/open-apis/base/v3/bases/%s/delete_auth_codes/validate"
)

type deleteApprovalSpec struct {
	Action       string
	BaseToken    string
	ResourceType string
	ResourceID   string
}

func deleteApprovalFlags() []common.Flag {
	return []common.Flag{
		{Name: "prepare-approval", Type: "bool", Desc: "create a delete approval request instead of deleting"},
		{Name: "auth-code", Desc: "one-time delete authorization code"},
	}
}

// handleDeleteApproval returns true when prepare mode handled the command and real deletion must stop.
func handleDeleteApproval(runtime *common.RuntimeContext, spec deleteApprovalSpec) (bool, error) {
	if runtime.Bool("prepare-approval") {
		result, err := baseV3Call(runtime, "POST", fmt.Sprintf(deleteApprovalRequestPath, spec.BaseToken), nil, map[string]interface{}{
			"action": spec.Action, "resource_type": spec.ResourceType, "resource_id": spec.ResourceID,
		})
		if err != nil {
			return false, err
		}
		runtime.Out(result, nil)
		return true, nil
	}
	if strings.TrimSpace(runtime.Str("auth-code")) == "" {
		return false, common.FlagErrorf("--auth-code is required; run the same command with --prepare-approval first")
	}
	_, err := baseV3Call(runtime, "POST", fmt.Sprintf(deleteAuthCodeValidatePath, spec.BaseToken), nil, map[string]interface{}{
		"auth_code": runtime.Str("auth-code"), "action": spec.Action,
		"resource_type": spec.ResourceType, "resource_id": spec.ResourceID,
	})
	if err != nil {
		return false, err
	}
	return false, nil
}
