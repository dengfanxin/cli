// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

// ── ProjectMemberAdd ────────────────────────────────────────────────────

var ProjectMemberAdd = common.Shortcut{
	Service:     "base",
	Command:     "+project-member-add",
	Description: "Add a member to the project",
	Risk:        "write",
	Scopes:      []string{"base:table:create", "base:field:create", "base:field:read", "base:field:update", "base:record:create"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "name", Desc: "member name", Required: true},
		{Name: "role", Desc: "member role", Required: true},
		{Name: "type", Desc: "member type", Default: "human", Enum: []string{"human", "agent"}},
		{Name: "user-id", Desc: "user ID (open_id or union_id)"},
	},
	DryRun: dryRunProjectMemberAdd,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectMemberAdd(runtime)
	},
}

func dryRunProjectMemberAdd(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find or create " + projectMembersTable).
		Set("base_token", bt).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("Create member record").
		Set("base_token", bt).
		Body(map[string]interface{}{
			"fields": map[string]interface{}{
				fieldMemberName:     runtime.Str("name"),
				fieldMemberRole:     runtime.Str("role"),
				fieldMemberType:     runtime.Str("type"),
				fieldMemberUserID:   runtime.Str("user-id"),
				fieldMemberJoinedAt: "now",
			},
		})
}

func executeProjectMemberAdd(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	name := strings.TrimSpace(runtime.Str("name"))
	role := strings.TrimSpace(runtime.Str("role"))
	memberType := strings.TrimSpace(runtime.Str("type"))
	if memberType == "" {
		memberType = "human"
	}
	userID := strings.TrimSpace(runtime.Str("user-id"))

	membersTableID, err := ensureProjectTable(runtime, baseToken, projectMembersTable)
	if err != nil {
		return err
	}

	fields := map[string]interface{}{
		fieldMemberName:     name,
		fieldMemberRole:     role,
		fieldMemberType:     memberType,
		fieldMemberJoinedAt: nowTimestamp(),
	}
	if userID != "" {
		fields[fieldMemberUserID] = userID
	}

	data, err := baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables", membersTableID, "records"), nil, fields)
	if err != nil {
		return err
	}

	runtime.Out(map[string]interface{}{"record": data, "created": true, "name": name}, nil)
	return nil
}

// ── ProjectMemberList ───────────────────────────────────────────────────

var ProjectMemberList = common.Shortcut{
	Service:     "base",
	Command:     "+project-member-list",
	Description: "List all project members",
	Risk:        "read",
	Scopes:      []string{"base:app:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
	},
	DryRun: dryRunProjectMemberList,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectMemberList(runtime)
	},
}

func dryRunProjectMemberList(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find " + projectMembersTable).
		Set("base_token", bt).
		GET("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("List member records").
		Set("base_token", bt)
}

func executeProjectMemberList(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)

	membersTableID, err := findProjectTableID(runtime, baseToken, projectMembersTable)
	if err != nil {
		return err
	}

	records, err := listAllRecords(runtime, baseToken, membersTableID)
	if err != nil {
		return err
	}

	items := make([]interface{}, 0, len(records))
	for _, r := range records {
		fields, _ := r["fields"].(map[string]interface{})
		entry := map[string]interface{}{
			"record_id": recordIDFromMap(r),
		}
		if fields != nil {
			entry[fieldMemberName] = fields[fieldMemberName]
			entry[fieldMemberRole] = fields[fieldMemberRole]
			entry[fieldMemberType] = fields[fieldMemberType]
			entry[fieldMemberUserID] = fields[fieldMemberUserID]
			entry[fieldMemberJoinedAt] = fields[fieldMemberJoinedAt]
		}
		items = append(items, entry)
	}

	runtime.Out(map[string]interface{}{"items": items, "count": len(items)}, nil)
	return nil
}
