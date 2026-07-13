// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseTableDelete = common.Shortcut{
	Service:     "base",
	Command:     "+table-delete",
	Description: "Delete a table by ID or name",
	Risk:        "high-risk-write",
	Scopes:      []string{"base:table:delete"},
	AuthTypes:   authTypes(),
	Flags:       append([]common.Flag{baseTokenFlag(true), tableRefFlag(true)}, deleteApprovalFlags()...),
	DryRun:      dryRunTableDelete,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeTableDelete(runtime)
	},
}
