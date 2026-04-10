# Project 项目协作指南

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

## 什么时候用 project 命令

用户说以下类似的话时，使用 project 命令：

- "建一个项目"、"帮我把项目搭起来"
- "把产品信息/需求/背景资料存到项目里"
- "看看项目现在什么情况"
- "给小张分配一个任务"、"帮我建几个任务"
- "看看有什么活要干"、"领个任务"
- "把这个经验/结论/接口定义记下来"
- "项目里有没有关于 xxx 的信息"

## 核心概念

**project-id = base-token**。一个项目就是一个多维表格，base-token 就是项目 ID。Agent 拿到一个 base-token 就能索引到项目的一切。

项目里有两类表：
- **骨架表**（`_project_info`、`_kv`、`_members`、`_tasks`、`_resources`）：由 project 命令自动管理，首次使用时按需创建
- **领域表**：用户根据业务需求自由创建的表，Agent 通过读取表结构理解项目的领域语言

## 意图识别与命令映射

### 创建项目

用户说"建个项目""搭个工作台""开个新项目"时：

```bash
lark-cli base +project-create --name "项目名" --description "一句话描述"
```

返回的 `project_id` 是后续一切操作的入口。**必须记住并告知用户。**

创建后，Agent 应主动把用户提到的背景信息（产品信息、目标受众、风格要求等）写入 KV，不需要用户明确说"写入 KV"。

### 了解项目

用户说"项目什么情况""进展怎样""帮我看看"时：

```bash
lark-cli base +project-get --base-token <project_id>
```

返回项目全貌：名称、成员、任务统计、KV 数量、自定义表。Agent 应该用自然语言总结，不要直接输出 JSON。

### 存储项目信息（KV）

用户提到产品信息、决策、经验、接口定义等需要记住的内容时，Agent 应主动写入 KV，不需要用户说"写 KV"：

```bash
# 写入
lark-cli base +project-kv-set --base-token <project_id> --key "自定义key" --value "内容" --source "来源"

# 读取
lark-cli base +project-kv-get --base-token <project_id> --key "key名"

# 列出（支持前缀过滤）
lark-cli base +project-kv-list --base-token <project_id> --prefix "前缀"

# 删除
lark-cli base +project-kv-delete --base-token <project_id> --key "key名"
```

**key 命名约定**：
- 产品信息：`product-*`
- 技术决策：`decision-*`
- 踩坑经验：`gotcha-*`
- 接口契约：`api-*`
- 工作产出：`output-*`

### 管理成员

用户说"把小张加进来""团队有谁"时：

```bash
# 添加
lark-cli base +project-member-add --base-token <project_id> --name "姓名" --role "角色" --type human

# 列出
lark-cli base +project-member-list --base-token <project_id>
```

type 只有两个值：`human` 或 `agent`。

### 管理资源

用户说"关联一下代码仓库""把设计文档加进来"时：

```bash
# 添加
lark-cli base +project-resource-add --base-token <project_id> --name "资源名" --type doc --url "链接"

# 列出
lark-cli base +project-resource-list --base-token <project_id>

# 移除
lark-cli base +project-resource-remove --base-token <project_id> --name "资源名"
```

type 常见值：`doc`（文档）、`repo`（代码仓库）、`drive`（云盘）、`wiki`（知识库）、`chat`（群聊）。

### 创建任务

用户说"建几个任务""把活分一下""脚本交给小张"时：

```bash
lark-cli base +project-task-add --base-token <project_id> \
  --title "任务标题" \
  --priority high \
  --assignee "负责人" \
  --description "详细描述" \
  --extra '{"type":"任务类型","其他自定义字段":"值"}'
```

**--extra 是关键**：任务表的列完全可以自定义。用户提到的任何分类维度（类型、模块、阶段、技能要求等），都可以通过 --extra 写入自定义字段。

priority 可选值：`high`、`medium`（默认）、`low`。

### 领取任务

用户说"看看有什么活""领个任务""把能做的做了"时：

```bash
# 领取（按优先级排序，自动标记为进行中）
lark-cli base +project-task-next --base-token <project_id>

# 按条件领取（只领自己能做的）
lark-cli base +project-task-next --base-token <project_id> --filter "type=脚本"
lark-cli base +project-task-next --base-token <project_id> --filter "assignee=小张"
```

--filter 支持多个条件（重复使用），按自定义字段过滤。

### 更新任务

任务做完或遇到阻塞时：

```bash
# 只更新状态和总结
lark-cli base +project-task-update --base-token <project_id> \
  --task-id <record_id> \
  --status done \
  --summary "一句话总结"

# 带结果数据（JSON 或纯文本，描述产出内容）
lark-cli base +project-task-update --base-token <project_id> \
  --task-id <record_id> \
  --status done \
  --summary "4个分镜脚本完成" \
  --result '[{"time":"0-5s","画面":"...","旁白":"..."},{"time":"5-15s",...}]'

# 带附件上传（多个 --attach 可以重复使用）
lark-cli base +project-task-update --base-token <project_id> \
  --task-id <record_id> \
  --status done \
  --summary "4张图生成完成" \
  --result '[{"name":"scene1.png","desc":"0-5s 地铁场景"},{"name":"scene2.png","desc":"5-15s 戴耳机"}]' \
  --attach ./scene1.png \
  --attach ./scene2.png \
  --attach ./scene3.png \
  --attach ./scene4.png
```

**三个字段的职责**：
- `summary`: 一句话总结，用于快速浏览任务列表
- `result`: 完整产出内容（JSON 或文本），下游 Agent 用来理解任务做了什么
- `attachments`: 实际文件（图片、视频、文档等），上传后挂在任务上

status 可选值：`pending`、`in_progress`、`done`、`blocked`。

### 读取任务详情

获取单个任务的完整信息（含 result 和 attachments）：

```bash
lark-cli base +project-task-get --base-token <project_id> --task-id <record_id>
```

返回：`title / status / priority / summary / task_result / attachments (含 file_token 和 url) / extras（自定义字段）`。

下游 Agent 拿到 attachments 后，可以用 `lark-cli drive +download --file-token <token>` 下载原文件到本地处理。

### 查看任务

```bash
# 所有任务
lark-cli base +project-task-list --base-token <project_id>

# 按状态过滤
lark-cli base +project-task-list --base-token <project_id> --status pending

# 按自定义字段过滤
lark-cli base +project-task-list --base-token <project_id> --filter "type=脚本"
```

### 监听任务（Worker Agent）

用户说"帮我盯着项目""有活就干""持续监听"时：

```bash
# 持续轮询，有匹配的 pending 任务就领取并输出
lark-cli base +project-task-listen --base-token <project_id> --filter "type=脚本" --interval 5

# 领取后自动调用 AI 执行（prompt 自动组装，含项目 KV + 任务详情）
lark-cli base +project-task-listen --base-token <project_id> --filter "type=脚本" --exec "claude -p"

# 只处理一个任务就退出
lark-cli base +project-task-listen --base-token <project_id> --filter "type=脚本" --exec "claude -p" --once
```

`--exec` 指定的命令通过 stdin 接收自动组装的 prompt（包含任务详情 + 项目 KV 数据）。执行成功后任务自动标记 done，失败则标记 blocked。

### 等待任务完成（Master Agent 调度）

用户说"等它做完再继续""做完了告诉我""等脚本写好了再做配图"时：

```bash
# 阻塞等待某个任务变成 done
lark-cli base +project-task-wait --base-token <project_id> --task-id <record_id>

# 等待其他状态
lark-cli base +project-task-wait --base-token <project_id> --task-id <record_id> --status blocked

# 设置超时（默认 600 秒）
lark-cli base +project-task-wait --base-token <project_id> --task-id <record_id> --timeout 300
```

**Master Agent 调度模式**：创建任务 → wait 等完成 → 创建下一个任务 → wait → ...以此类推。Master 掌握全局流程，Worker Agent 各自 listen 领取属于自己的任务。

## Agent 行为规范

### 创建项目时

1. 从用户的描述中提取：项目名称、项目描述、产品信息、目标受众、风格要求等
2. 调用 `+project-create` 创建项目
3. 把背景信息**主动**写入 KV（用户不需要说"写 KV"）
4. 告诉用户 project_id，提示他可以分享给团队

### 作为 Master Agent 调度时

Master Agent 负责全局流程编排，不做具体执行：

1. 理解用户需求，拆解成任务序列
2. 按顺序创建任务：`+project-task-add --extra '{"type":"脚本","agent":"小张"}'`
3. 等待完成：`+project-task-wait --task-id <id>`
4. 看到 done 后创建下一个任务
5. 全部完成后汇总结果通知用户

### 作为 Worker Agent 工作时

Worker Agent 负责具体执行，不关心全局流程：

1. **`+project-get`** 了解项目全貌
2. **`+project-kv-list`** 看看有什么已有信息
3. **领取任务** 时用 `--filter` 只领自己该做的
4. **做完任务** 后：
   - 把产出写入 KV（供下游 Agent 读取）
   - 更新任务状态为 done
   - summary 里写清楚产出在哪个 KV key
5. 或者直接用 **`+project-task-listen --exec`** 自动循环

### 多 Agent 协作模式

Agent 之间不直接通信。协作通过三个机制：

- **Task**：协作协议。Master 创建任务，Worker 领取执行
- **KV**：数据传递。上游 Agent 把产出写入 KV，下游 Agent 按 key 读取
- **Wait**：顺序控制。Master 用 task-wait 确保上游完成后再创建下游任务

示例流程：
```
Master Agent:
  task-add "脚本" → task-wait → task-add "配图" → task-wait → task-add "合成" → ...

Worker Agent (小张):                    Worker Agent (小李):
  task-listen --filter "type=脚本"        task-listen --filter "type=合成"
  → 领取 → 写脚本 → kv-set "script-1"     → 领取 → 读 kv "script-1" + "images-1"
  → task-update done                       → 合成视频 → kv-set "video-raw-1"
  → 领取 "配图" → 读 kv "script-1"          → task-update done
  → 画配图 → kv-set "images-1"
  → task-update done
```

## 命令速查

| 意图 | 命令 |
|------|------|
| 创建项目 | `+project-create --name --description` |
| 查看项目 | `+project-get --base-token` |
| 写入信息 | `+project-kv-set --base-token --key --value` |
| 读取信息 | `+project-kv-get --base-token --key` |
| 列出信息 | `+project-kv-list --base-token [--prefix]` |
| 添加成员 | `+project-member-add --base-token --name --role` |
| 列出成员 | `+project-member-list --base-token` |
| 添加资源 | `+project-resource-add --base-token --name --type --url` |
| 列出资源 | `+project-resource-list --base-token` |
| 创建任务 | `+project-task-add --base-token --title [--extra]` |
| 领取任务 | `+project-task-next --base-token [--filter]` |
| 更新任务 | `+project-task-update --base-token --task-id [--status --summary --result --attach]` |
| 查看任务列表 | `+project-task-list --base-token [--status] [--filter]` |
| 获取单个任务 | `+project-task-get --base-token --task-id` |
| 监听任务 | `+project-task-listen --base-token [--filter] [--exec]` |
| 等待任务完成 | `+project-task-wait --base-token --task-id [--timeout]` |
