// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"
	"time"

	"github.com/larksuite/cli/shortcuts/common"
)

var ProjectCreate = common.Shortcut{
	Service:     "base",
	Command:     "+project-create",
	Description: "Create a new project (base) with project info table",
	Risk:        "write",
	Scopes:      []string{"base:app:create"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		{Name: "name", Desc: "project name", Required: true},
		{Name: "description", Desc: "project description"},
		{Name: "folder-token", Desc: "folder token for destination"},
	},
	DryRun: dryRunProjectCreate,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectCreate(runtime)
	},
}

func dryRunProjectCreate(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	body := map[string]interface{}{"name": runtime.Str("name")}
	if folderToken := strings.TrimSpace(runtime.Str("folder-token")); folderToken != "" {
		body["folder_token"] = folderToken
	}
	return common.NewDryRunAPI().
		POST("/open-apis/base/v3/bases").
		Desc("Create base for project").
		Body(body).
		POST("/open-apis/base/v3/bases/:base_token/tables").
		Desc("Create _project_info table").
		Body(map[string]interface{}{"name": projectInfoTable}).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("Create initial project info record").
		Body(map[string]interface{}{
			"fields": map[string]interface{}{
				fieldInfoName:        runtime.Str("name"),
				fieldInfoDescription: runtime.Str("description"),
				fieldInfoStatus:      "active",
			},
		})
}

func executeProjectCreate(runtime *common.RuntimeContext) error {
	name := strings.TrimSpace(runtime.Str("name"))
	description := strings.TrimSpace(runtime.Str("description"))

	// Step 1: Create the base.
	body := map[string]interface{}{"name": name}
	if folderToken := strings.TrimSpace(runtime.Str("folder-token")); folderToken != "" {
		body["folder_token"] = folderToken
	}
	baseData, err := baseV3Call(runtime, "POST", baseV3Path("bases"), nil, body)
	if err != nil {
		return err
	}
	baseToken, _ := baseData["app_token"].(string)
	if baseToken == "" {
		baseToken, _ = baseData["base_token"].(string)
	}
	if baseToken == "" {
		if app, ok := baseData["app"].(map[string]interface{}); ok {
			baseToken, _ = app["app_token"].(string)
		}
	}

	// Step 2: Create the _project_info table directly (no need to list tables
	// since we just created the base and know it's empty).
	tableData, err := baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables"), nil, map[string]interface{}{"name": projectInfoTable})
	if err != nil {
		return err
	}
	infoTableID := tableID(tableData)

	// Step 3: Create fields for the info table.
	// The default table only has an auto-number "ID" field; we need to add our fields.
	infoFields := []string{fieldInfoName, fieldInfoDescription, fieldInfoStatus}
	for i, fieldName := range infoFields {
		_, err = baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables", infoTableID, "fields"), nil, map[string]interface{}{
			"type": "text",
			"name": fieldName,
		})
		if err != nil {
			return err
		}
		if i < len(infoFields)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Step 4: Create the initial info record.
	infoRecord := map[string]interface{}{
		fieldInfoName:        name,
		fieldInfoDescription: description,
		fieldInfoStatus:      "active",
	}
	_, err = baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables", infoTableID, "records"), nil, infoRecord)
	if err != nil {
		return err
	}

	runtime.Out(map[string]interface{}{
		"created":    true,
		"project_id": baseToken,
		"name":       name,
	}, nil)
	return nil
}
