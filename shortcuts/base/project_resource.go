// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

// ── ProjectResourceAdd ─────────────────────────────────────────────────

var ProjectResourceAdd = common.Shortcut{
	Service:     "base",
	Command:     "+project-resource-add",
	Description: "Add a resource to the project",
	Risk:        "write",
	Scopes:      []string{"base:table:create", "base:field:create", "base:field:read", "base:field:update", "base:record:create"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "name", Desc: "resource name", Required: true},
		{Name: "type", Desc: "resource type (doc/repo/drive/wiki/chat)", Required: true},
		{Name: "url", Desc: "resource URL", Required: true},
		{Name: "description", Desc: "resource description"},
	},
	DryRun: dryRunProjectResourceAdd,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectResourceAdd(runtime)
	},
}

func dryRunProjectResourceAdd(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find or create " + projectResourcesTable).
		Set("base_token", bt).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("Create resource record").
		Set("base_token", bt).
		Body(map[string]interface{}{
			"fields": map[string]interface{}{
				fieldResourceName:        runtime.Str("name"),
				fieldResourceType:        runtime.Str("type"),
				fieldResourceURL:         runtime.Str("url"),
				fieldResourceDescription: runtime.Str("description"),
			},
		})
}

func executeProjectResourceAdd(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	name := strings.TrimSpace(runtime.Str("name"))
	resType := strings.TrimSpace(runtime.Str("type"))
	url := strings.TrimSpace(runtime.Str("url"))
	description := strings.TrimSpace(runtime.Str("description"))

	resourcesTableID, err := ensureProjectTable(runtime, baseToken, projectResourcesTable)
	if err != nil {
		return err
	}

	fields := map[string]interface{}{
		fieldResourceName: name,
		fieldResourceType: resType,
		fieldResourceURL:  url,
	}
	if description != "" {
		fields[fieldResourceDescription] = description
	}

	data, err := baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables", resourcesTableID, "records"), nil, fields)
	if err != nil {
		return err
	}

	runtime.Out(map[string]interface{}{"record": data, "created": true, "name": name}, nil)
	return nil
}

// ── ProjectResourceList ────────────────────────────────────────────────

var ProjectResourceList = common.Shortcut{
	Service:     "base",
	Command:     "+project-resource-list",
	Description: "List all project resources",
	Risk:        "read",
	Scopes:      []string{"base:app:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
	},
	DryRun: dryRunProjectResourceList,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectResourceList(runtime)
	},
}

func dryRunProjectResourceList(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find " + projectResourcesTable).
		Set("base_token", bt).
		GET("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("List resource records").
		Set("base_token", bt)
}

func executeProjectResourceList(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)

	resourcesTableID, err := findProjectTableID(runtime, baseToken, projectResourcesTable)
	if err != nil {
		return err
	}

	records, err := listAllRecords(runtime, baseToken, resourcesTableID)
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
			entry[fieldResourceName] = fields[fieldResourceName]
			entry[fieldResourceType] = fields[fieldResourceType]
			entry[fieldResourceURL] = fields[fieldResourceURL]
			entry[fieldResourceDescription] = fields[fieldResourceDescription]
		}
		items = append(items, entry)
	}

	runtime.Out(map[string]interface{}{"items": items, "count": len(items)}, nil)
	return nil
}

// ── ProjectResourceRemove ──────────────────────────────────────────────

var ProjectResourceRemove = common.Shortcut{
	Service:     "base",
	Command:     "+project-resource-remove",
	Description: "Remove a resource from the project by name",
	Risk:        "write",
	Scopes:      []string{"base:record:delete"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "name", Desc: "resource name to remove", Required: true},
	},
	DryRun: dryRunProjectResourceRemove,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectResourceRemove(runtime)
	},
}

func dryRunProjectResourceRemove(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find " + projectResourcesTable).
		Set("base_token", bt).
		GET("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("List records to find resource by name").
		Set("base_token", bt).
		DELETE("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/:record_id").
		Desc("Delete the resource record").
		Set("base_token", bt)
}

func executeProjectResourceRemove(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	name := strings.TrimSpace(runtime.Str("name"))

	resourcesTableID, err := findProjectTableID(runtime, baseToken, projectResourcesTable)
	if err != nil {
		return err
	}

	records, err := searchRecordsByField(runtime, baseToken, resourcesTableID, fieldResourceName, name)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return fmt.Errorf("resource %q not found", name)
	}

	recordID := recordIDFromMap(records[0])
	_, err = baseV3Call(runtime, "DELETE", baseV3Path("bases", baseToken, "tables", resourcesTableID, "records", recordID), nil, nil)
	if err != nil {
		return err
	}

	runtime.Out(map[string]interface{}{"deleted": true, "name": name}, nil)
	return nil
}
