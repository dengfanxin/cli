// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var ProjectGet = common.Shortcut{
	Service:     "base",
	Command:     "+project-get",
	Description: "Get project overview: info, members, tasks, resources, and custom tables",
	Risk:        "read",
	Scopes:      []string{"base:app:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
	},
	DryRun: dryRunProjectGet,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectGet(runtime)
	},
}

func dryRunProjectGet(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token").
		Desc("Get base info").
		Set("base_token", bt).
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List all tables").
		Set("base_token", bt)
}

func executeProjectGet(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)

	// Get base info.
	baseData, err := baseV3Call(runtime, "GET", baseV3Path("bases", baseToken), nil, nil)
	if err != nil {
		return err
	}

	// List all tables.
	tables, err := listAllProjectTables(runtime, baseToken)
	if err != nil {
		return err
	}

	result := map[string]interface{}{
		"project_id": baseToken,
		"base":       baseData,
	}

	var customTables []interface{}

	for _, t := range tables {
		tName := tableNameFromMap(t)
		tID := tableID(t)

		if !isWellKnownTable(tName) {
			customTables = append(customTables, map[string]interface{}{
				"table_id":   tID,
				"table_name": tName,
			})
			continue
		}

		switch tName {
		case projectInfoTable:
			records, err := listAllRecords(runtime, baseToken, tID)
			if err != nil {
				result["info_error"] = err.Error()
				continue
			}
			if len(records) > 0 {
				fields, _ := records[0]["fields"].(map[string]interface{})
				result["info"] = fields
			}

		case projectMembersTable:
			records, err := listAllRecords(runtime, baseToken, tID)
			if err != nil {
				result["members_error"] = err.Error()
				continue
			}
			members := make([]interface{}, 0, len(records))
			for _, r := range records {
				fields, _ := r["fields"].(map[string]interface{})
				members = append(members, fields)
			}
			result["members"] = members

		case projectKVTable:
			_, total, err := listProjectRecords(runtime, baseToken, tID)
			if err != nil {
				result["kv_error"] = err.Error()
				continue
			}
			result["kv_count"] = total

		case projectTasksTable:
			records, total, err := listProjectRecords(runtime, baseToken, tID)
			if err != nil {
				result["tasks_error"] = err.Error()
				continue
			}
			stats := map[string]interface{}{
				"total":       total,
				"pending":     0,
				"in_progress": 0,
				"done":        0,
				"blocked":     0,
			}
			for _, r := range records {
				status := recordFieldValue(r, fieldTaskStatus)
				if cur, ok := stats[status].(int); ok {
					stats[status] = cur + 1
				}
			}
			result["tasks"] = stats

		case projectResourcesTable:
			records, err := listAllRecords(runtime, baseToken, tID)
			if err != nil {
				result["resources_error"] = err.Error()
				continue
			}
			resources := make([]interface{}, 0, len(records))
			for _, r := range records {
				fields, _ := r["fields"].(map[string]interface{})
				resources = append(resources, fields)
			}
			result["resources"] = resources
		}
	}

	if len(customTables) > 0 {
		result["custom_tables"] = customTables
	}

	runtime.Out(result, nil)
	return nil
}
