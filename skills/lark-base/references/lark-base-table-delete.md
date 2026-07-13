# base +table-delete

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

删除一张表。

## 推荐命令

```bash
lark-cli base +table-delete \
  --base-token app_xxx \
  --table-id tbl_xxx \
  --auth-code larkauth_v1_xxx \
  --yes
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--base-token <token>` | 是 | Base Token |
| `--table-id <id_or_name>` | 是 | 表 ID 或表名 |
| `--prepare-approval` | 否 | 只创建审批请求并返回 approval_url / request_id |
| `--auth-code <code>` | 执行删除时是 | 用户从可信审批页获取的一次性授权码 |

## API 入参详情

**HTTP 方法和路径：**

```
DELETE /open-apis/base/v3/bases/:base_token/tables/:table_id
```

## 返回重点

- 返回 `deleted: true` 以及删除目标的 `table_id / table_name`。

## 工作流

> 这是**高风险写入操作**。CLI 层要求显式传 `--yes`；如果用户已经明确要求删除且目标明确，直接执行并带上 `--yes`，不要再补一次确认。

1. 先用同一命令加 `--prepare-approval` 创建审批请求并把链接交给用户。
2. 用户回填授权码后，带 `--auth-code` 和 `--yes` 执行一次真实 DELETE。

## 坑点

- ⚠️ 高风险不可逆操作。
- ⚠️ 删除场景强烈建议传 `tbl_xxx`，不要传表名。
- ⚠️ 忘记带 `--yes` 会被 CLI 拦截。

## 参考

- [lark-base-table.md](lark-base-table.md) — table 索引页
- [lark-base-table-list.md](lark-base-table-list.md) — 列表
