---
name: lark-multi-agent-collab
version: 1.0.0
description: "多智能体协作框架：把 lark-cli 的 project 命令封装成一套通用的 Master-Worker 协作模式，让任意多个 Agent 能够围绕一个项目分工协作。当用户说 '你是 master/worker'、'带领 xx agent 干活'、'你擅长 xx 任务'、'多智能体协作' 时使用。用户可以通过自然语言声明 Agent 角色和专长，skill 负责环境初始化、角色初始化、任务协议、交接协议。"
metadata:
  requires:
    bins: ["lark-cli"]
---

# Multi-Agent Collaboration (v1)

**CRITICAL — 开始前 MUST 先用 Read 工具读取这几个文件：**
1. [`../lark-shared/SKILL.md`](../lark-shared/SKILL.md) — 认证和全局参数
2. [`../lark-base/SKILL.md`](../lark-base/SKILL.md) — Base 命令参考
3. [`../lark-base/references/lark-base-project-guide.md`](../lark-base/references/lark-base-project-guide.md) — project 命令完整说明

本 skill **不重复**上述文档中的命令语法，只定义多 Agent 协作的模式和角色职责。

---

## 核心理念

**把 Base 当作 Agent 之间的共享状态，每个 Agent 只管自己的活。**

- **协作协议**：Task 表（谁做什么、做到哪了、下一步谁接）
- **数据共享**：KV 表（跨 Agent 传递小数据）+ Task Attachments（跨 Agent 传递文件）
- **顺序控制**：Master Agent 通过 `+project-task-wait` 串行编排
- **并行工作**：每个 Worker Agent 通过 `+project-task-next --filter` 只领自己专长的任务

Agent 之间**不直接通信**——所有协作通过 Base 的读写完成。

---

## 角色模型

每个 Agent 扮演两种角色之一：

| 角色 | 职责 |
|------|------|
| **Master** | 理解需求、拆解任务、创建任务、监控进度、汇总结果、通知用户 |
| **Worker** | 按专长领取任务、读取上游产出、执行具体工作、写入产出、更新状态 |

一个项目里有 **1 个 Master** 和 **N 个 Worker**。Master 不执行具体工作，Worker 不关心全局流程。

---

## 用户调用方式

用户通过自然语言声明 Agent 的角色和专长：

**声明 Master**：
> 你是 master，你负责安排和搞定视频生成任务，带领 a、b、c 三个 agent 干活，a 擅长写电影脚本，b 擅长作画，c 擅长生成视频

**声明 Worker**：
> 你是 a，你擅长写电影脚本
> 你是 b，你擅长 AI 作画，用 nanobanana 2
> 你是 c，你擅长用 ffmpeg 做视频合成

Agent 收到这种声明后，执行对应角色的初始化流程（见下文）。

---

## Master 初始化流程

当用户说 "你是 master ..." 时：

### 1. 解析角色声明

从用户的话里提取：
- **自己的角色名**：`master` 或用户指定的名字（如"梵助手"）
- **项目目标**：要搞定的事情（如"视频生成任务"）
- **Worker 列表**：每个 Worker 的名字 + 专长
- **任务类型映射**：根据专长推导出 task.type 的值

例如上面的声明会被解析成：

```yaml
role: master
goal: 视频生成任务
workers:
  - name: a
    skill: 写电影脚本
    task_type: 脚本
  - name: b
    skill: AI 作画
    task_type: 配图
  - name: c
    skill: 视频合成
    task_type: 视频
```

### 2. 环境检查

检查 lark-cli 是否已安装并通过认证。如未安装或未授权，引导用户完成：
- 参考 `../lark-shared/SKILL.md` 的安装和认证流程
- 需要 base 相关 scope：`base:app:create base:app:read base:table:create base:record:*  base:field:*`

### 3. 创建或定位项目

```
如果用户没有给 project_id：
  → 用 +project-create 创建新项目，名称用 "goal + 日期"
  → 把项目 ID 回告用户，让用户分享给 Worker Agents

如果用户给了 project_id：
  → 用 +project-get 验证可访问
```

### 4. 把角色定义写入项目 KV

用约定的 key 存储 Agent 拓扑，让 Worker 初始化时可以读到：

```
kv key "roles/master"   → master 名字 + 项目目标
kv key "roles/workers"  → workers JSON 数组（name/skill/task_type）
```

### 5. 把关键背景信息写入 KV

用户提到的任何背景信息（产品信息、品牌调性、参考资料）主动写入 KV：

```
kv key "context/product"    → 产品信息
kv key "context/brand"      → 品牌/风格要求
kv key "context/audience"   → 目标人群
kv key "context/..."        → 其他
```

### 6. 告知用户下一步

```
✅ Master 已就绪
项目 ID: appXXX
告诉 Worker Agents：
  "你是 a（脚本），项目 ID 是 appXXX"
  "你是 b（配图），项目 ID 是 appXXX"
  "你是 c（视频），项目 ID 是 appXXX"
等 Worker 都就绪后，告诉我具体的任务需求。
```

### 7. 接收任务需求并编排

当用户说具体需求时（如"做一条 30 秒的 Ola 耳机广告"）：

```
循环:
  1. 确定下一步该做什么（根据 Worker 的专长）
  2. +project-task-add --extra '{"type":"<task_type>","assignee":"<worker_name>"}'
     注意 extra 里必须带 type 和 assignee，这是 Worker 领取的依据
  3. +project-task-wait --task-id <id>  (阻塞等 Worker 完成)
  4. +project-task-get --task-id <id>   (读取产出)
  5. 把下游需要的上游产出写入 KV 或留在 task attachments 里
  6. 创建下一个任务...
直到全部完成
```

### 8. 验收并汇报

所有任务 done 后：
- 用 `+project-get` 汇总统计
- 用 `+project-task-list --status done` 列出所有产出
- 用自然语言向用户报告结果，附上关键产物的位置（task_id 或 KV key）

---

## Worker 初始化流程

当用户说 "你是 xxx，你擅长 ..." 时：

### 1. 解析角色声明

提取：
- **自己的 worker 名字**：如 `a`
- **专长描述**：如"写电影脚本"
- **推导 task_type**：对应 Master 拆解任务时使用的类型（询问用户确认或从项目 KV 读取）

### 2. 环境检查 + 专长工具检查

除了 lark-cli 环境（参考 `../lark-shared/SKILL.md`），还要检查专长所需的工具：

| 专长类型 | 需要的工具 | 检查方式 |
|---------|-----------|---------|
| 写脚本/文案 | 不需要额外工具 | - |
| AI 作画 | 图片生成模型 API | 询问用户 API 可用性 |
| 视频合成 | 视频合成工具链 | 询问用户 |
| 视频后期 | ffmpeg | `which ffmpeg` |
| 代码实现 | 对应语言工具链 | - |

**没有的工具要引导用户安装**，不要自己瞎装。

### 3. 获取项目上下文

```
1. +project-get --base-token <project_id>
   → 了解项目目标、成员、任务概况
2. +project-kv-list --prefix "context/"
   → 读取所有背景信息（产品、品牌、人群等）
3. +project-kv-get --key "roles/workers"
   → 确认自己在 worker 列表中，获取准确的 task_type
```

### 4. 进入工作循环

两种模式，优先选第二种：

**模式 A：手动领取（调试时用）**
```
+project-task-next --filter "type=<自己的task_type>"
→ 执行工作
→ +project-task-update --status done --summary ... --result ... --attach <file>
```

**模式 B：自动监听（生产环境）**
```
+project-task-listen --filter "type=<自己的task_type>" --exec "<自己的 agent 命令>"
```

`--exec` 指定的命令会接收到自动组装的 prompt（含项目 KV 和任务详情），执行完自动标记 done。

### 5. 告知用户就绪

```
✅ Worker <name> 已就绪
专长: <skill>
任务类型过滤: type=<task_type>
项目: <project_name> (<project_id>)
现在开始监听任务...
```

---

## 任务协作协议

所有 Agent 必须遵守以下协议：

### 创建任务（Master）

每个任务**必须**带以下字段：

| 字段 | 说明 |
|------|------|
| `title` | 任务标题，简短清晰 |
| `description` | 详细描述，Worker 依此理解要做什么 |
| `priority` | `high` / `medium` / `low` |
| `extra.type` | **必须**，Worker 靠这个过滤领取 |
| `extra.assignee` | Worker 名字（软约定，type 过滤是硬约束） |

可选：
- `extra.upstream_task_ids` — 依赖的上游任务 ID 列表，下游 Worker 用这个找上游产出
- `extra.output_key` — 约定产出写入的 KV key（如果不用 attachment）

### 执行任务（Worker）

领取到任务后：

1. **读取任务详情**：`+project-task-get --task-id <id>` 获取完整字段
2. **读取上游产出**：
   - 如果 `extra.upstream_task_ids` 存在，对每个上游 `task-get` 读 attachments 和 result
   - 如果任务需要项目级信息，读 `kv-get` 相应 key
3. **执行具体工作**（调用自己的专长工具链）
4. **更新任务状态**：
   ```
   +project-task-update \
     --task-id <id> \
     --status done \
     --summary "一句话总结" \
     --result '<JSON 描述产出>' \
     --attach <file1> --attach <file2> ...
   ```

### 交接约定

- **有文件产出** → 用 `--attach` 挂到当前任务上，同时用 `--result` 写元数据（每个文件是什么）
- **只有数据产出** → 用 `--result` 写 JSON
- **纯文本产出**（脚本、文案）→ 用 `--result` 写文本

下游 Worker 读 `task-get` 即可拿到 `attachments`（含 file_token）和 `task_result`。

### 阻塞处理

如果 Worker 无法完成任务：

```
+project-task-update --task-id <id> --status blocked \
  --summary "阻塞原因：<具体说明>"
```

Master 的 `+project-task-wait` 默认等 `done`，遇到 `blocked` 会收到警告，然后决定：
- 重新创建任务（换一种做法）
- 向用户报告需要人工介入

---

## 跨 Agent 协作模板

### 三角色流水线（常见场景）

```
Master: 创建任务 A → wait → 创建任务 B（依赖 A） → wait → 创建任务 C（依赖 B） → wait → 汇报
Worker a: task-listen --filter "type=A的类型" → 执行 → 更新
Worker b: task-listen --filter "type=B的类型" → 执行（读 A 的产出） → 更新
Worker c: task-listen --filter "type=C的类型" → 执行（读 B 的产出） → 更新
```

### 扇出扇入（一个人产出，多个人消费）

```
Master: 创建任务 A → wait → 并行创建任务 B1、B2、B3（都依赖 A） → 分别 wait → 创建任务 C（依赖 B1/B2/B3）
```

B1/B2/B3 的 `extra.upstream_task_ids` 都指向 A 的 task_id，它们读同一份上游产出做不同的工作。

---

## 常见陷阱

1. **Worker 不带 `--filter` 领任务**：会抢到不属于自己的任务。必须严格按 `task_type` 过滤。
2. **Master 没等就创建下一个任务**：下游 Worker 读不到上游产出。必须用 `task-wait`。
3. **产出没写入 task 或 KV**：下游看不到，等于没做。更新任务时 `--result` 或 `--attach` 必须带。
4. **task_type 不一致**：Master 创建时写 `"视频生成"`，Worker 过滤 `"视频"`，永远领不到。Master 写入 `roles/workers` KV 时要明确约定。
5. **把 Worker 的工具安装和 lark-cli 安装搞混**：Worker 的专长工具（ffmpeg、绘图模型 API 等）是另一回事，本 skill 不负责，参考 `../lark-shared/SKILL.md` 只处理 lark-cli 环境。

---

## 参考

- [`../lark-shared/SKILL.md`](../lark-shared/SKILL.md) — 认证、安装、全局参数
- [`../lark-base/SKILL.md`](../lark-base/SKILL.md) — Base 命令索引
- [`../lark-base/references/lark-base-project-guide.md`](../lark-base/references/lark-base-project-guide.md) — project 命令完整语法和 flags
