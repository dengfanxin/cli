// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var ProjectTaskGet = common.Shortcut{
	Service:     "base",
	Command:     "+project-task-get",
	Description: "Get a single task with all fields including result and attachments",
	Risk:        "read",
	Scopes:      []string{"base:app:read", "base:record:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "task-id", Desc: "record ID of the task", Required: true},
	},
	DryRun: func(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		return common.NewDryRunAPI().
			GET("/open-apis/base/v3/bases/:base_token/tables").
			Desc("List tables to find " + projectTasksTable).
			Set("base_token", projectBaseToken(runtime)).
			GET("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/:record_id").
			Desc("Read task record").
			Set("base_token", projectBaseToken(runtime)).
			Set("record_id", runtime.Str("task-id"))
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectTaskGet(runtime)
	},
}

func executeProjectTaskGet(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	taskID := runtime.Str("task-id")

	tasksTableID, err := findProjectTableID(runtime, baseToken, projectTasksTable)
	if err != nil {
		return err
	}

	data, err := baseV3Call(runtime, "GET",
		baseV3Path("bases", baseToken, "tables", tasksTableID, "records", taskID),
		nil, nil)
	if err != nil {
		return err
	}

	// Parse fields from response. Base v3 uses compact format:
	// {"data": [[val1, val2, ...]], "fields": ["f1", "f2", ...], "record_id_list": [...]}
	// or standard format: {"fields": {...}, "record_id": "..."}
	result := map[string]interface{}{
		"task_id": taskID,
	}

	// Try standard format first.
	if fields, ok := data["fields"].(map[string]interface{}); ok && fields != nil {
		populateTaskResult(result, fields)
		runtime.Out(result, nil)
		return nil
	}

	// Compact format.
	fieldNames := toStringSlice(data["fields"])
	rawRows, _ := data["data"].([]interface{})
	if len(fieldNames) > 0 && len(rawRows) > 0 {
		row, _ := rawRows[0].([]interface{})
		fields := map[string]interface{}{}
		for i, col := range row {
			if i < len(fieldNames) {
				fields[fieldNames[i]] = col
			}
		}
		populateTaskResult(result, fields)
	}

	runtime.Out(result, nil)
	return nil
}

// populateTaskResult extracts task fields into the result map, handling
// attachments specially to expose download URLs.
func populateTaskResult(result map[string]interface{}, fields map[string]interface{}) {
	result["title"] = fields[fieldTaskTitle]
	result["description"] = fields[fieldTaskDescription]
	result["status"] = fields[fieldTaskStatus]
	result["priority"] = fields[fieldTaskPriority]
	result["assignee"] = fields[fieldTaskAssignee]
	result["summary"] = fields[fieldTaskSummary]
	result["task_result"] = fields[fieldTaskResult]
	result["created_at"] = fields[fieldTaskCreatedAt]
	result["updated_at"] = fields[fieldTaskUpdatedAt]

	// Attachments: extract file_token, name, and url.
	if rawAttachments, ok := fields[fieldTaskAttachments].([]interface{}); ok {
		attachments := make([]map[string]interface{}, 0, len(rawAttachments))
		for _, item := range rawAttachments {
			if m, ok := item.(map[string]interface{}); ok {
				att := map[string]interface{}{}
				if ft, _ := m["file_token"].(string); ft != "" {
					att["file_token"] = ft
				}
				if name, _ := m["name"].(string); name != "" {
					att["name"] = name
				}
				if mimeType, _ := m["mime_type"].(string); mimeType != "" {
					att["mime_type"] = mimeType
				}
				if size := m["size"]; size != nil {
					att["size"] = size
				}
				if url, _ := m["url"].(string); url != "" {
					att["url"] = url
				}
				if tmpURL, _ := m["tmp_url"].(string); tmpURL != "" {
					att["tmp_url"] = tmpURL
				}
				attachments = append(attachments, att)
			}
		}
		result["attachments"] = attachments
	}

	// Include any custom fields (extras).
	for k, v := range fields {
		if !knownTaskFields[k] && k != "ID" {
			if result["extras"] == nil {
				result["extras"] = map[string]interface{}{}
			}
			result["extras"].(map[string]interface{})[k] = v
		}
	}
}
