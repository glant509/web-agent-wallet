# web3-service-agent 启动过程与运行流程分析

## 1. 总览

这个工程当前可以看作一个最小可运行的 Agent Runtime，核心链路是：

```text
application.properties
        │
        ▼
     Config.Load
        │
        ▼
  Session / Prompt / Tool Registry / LLM Client
        │
        ▼
    Agent Runtime
        │
        ▼
     HTTP API
        │
        ├─ POST /v1/sessions
        ├─ POST /v1/agent/runs
        └─ POST /v1/agent/runs/stream
```

启动时主要做依赖装配；运行时主要做 Session 管理、Prompt 构造、模型调用、Tool Calling 闭环。

---

## 2. 启动入口

入口在：

- `cmd/web3-service-agent/main.go`

`main()` 的启动步骤是固定顺序的。

### 2.1 读取配置

先执行：

```go
cfg, err := config.Load()
```

配置文件默认来自：

- `etc/application.properties`

这里会完成两件事：

1. 读取 `service.*`
2. 读取 `models.*`

然后根据：

- `service.modelProvider`
- `service.modelName`

在 `models` 列表里选出一个 `used=true` 的模型作为当前生效模型。

如果没找到、被禁用、或者配置不完整，启动阶段就会失败。

### 2.2 创建基础组件

接着初始化：

```go
sessionManager := session.NewManager()
promptBuilder := prompt.NewBuilder(cfg.Service.Name)
toolRegistry := tool.NewRegistry()
mcpRegistry := mcp.NewRegistry()
```

它们的职责分别是：

| 组件 | 作用 |
| --- | --- |
| `session.Manager` | 保存会话历史消息 |
| `prompt.Builder` | 生成 system prompt，并把工具列表注入进去 |
| `tool.Registry` | 注册工具、输出工具定义、执行工具 |
| `mcp.Registry` | 目前只是预留，启动后没有真正接入运行链路 |

### 2.3 注册工具

启动时会执行：

```go
tools.RegisterAll(toolRegistry)
```

入口在：

- `tools/register.go`

它会依次调用各子目录的 `Register(...)`。当前更合理的方向是按能力分类：

- `market`
- `dex`
- `bridge`
- `yield`

当前真正有可用实现的主要是：

- `tools/market/register.go`
  - 注册了 `market_token_overview`

其它大多数工具目录目前还是空注册。

### 2.4 选择 LLM Client

然后执行：

```go
llmClient, err := provider.NewClient(&cfg)
```

位置：

- `internal/llm/provider/factory.go`

这里根据 `service.modelProvider` 选择 provider。

当前支持：

- `openai`
- `openrouter`
- `deepseek`

这三种 provider 现在都走同一套 OpenAI-compatible chat completions 客户端：

- `internal/llm/openai/client.go`

也就是说：

- OpenAI：走 `/chat/completions`
- OpenRouter：走 `/chat/completions`
- DeepSeek：走 `/chat/completions`

只是：

- `base_url`
- `api_key`
- `model name`

来自不同配置。

### 2.5 装配 Agent Runtime

执行：

```go
runtime := agent.New(agent.Dependencies{...})
```

位置：

- `internal/agent/runtime.go`

此时 Runtime 会拿到：

- Session 管理器
- Prompt Builder
- LLM Client
- Tool Registry
- MaxSteps

这一步完成后，Agent 的核心闭环已经具备。

### 2.6 暴露 HTTP 服务

执行：

```go
server := &http.Server{
    Addr: cfg.Service.ListenAddr(),
    Handler: httpapi.New(runtime, sessionManager),
}
```

HTTP 路由在：

- `internal/httpapi/server.go`

注册了 4 个接口：

| 路径 | 作用 |
| --- | --- |
| `GET /healthz` | 健康检查 |
| `POST /v1/sessions` | 创建会话 |
| `POST /v1/agent/runs` | 同步执行一次完整 Agent Run |
| `POST /v1/agent/runs/stream` | 流式执行 |

### 2.7 启动与优雅退出

最后：

1. 用 goroutine 启动 `ListenAndServe()`
2. 用 `signal.NotifyContext` 监听 `SIGINT` / `SIGTERM`
3. 收到退出信号后调用 `server.Shutdown(...)`

所以当前 HTTP 服务器是支持优雅关闭的。

---

## 3. 配置加载流程

位置：

- `internal/config/config.go`

### 3.1 文件读取

`Load()` -> `LoadFrom(defaultConfigFile)` -> `readProperties(path)`

读取逻辑是 Java properties 风格：

- 支持 `=` / `:`
- 跳过空行
- 跳过 `#` / `!` 注释

### 3.2 环境变量展开

每个值都会经过：

```go
resolveValue(value)
```

内部使用：

```go
os.Expand(...)
```

所以像：

```properties
models.3.api_key=${DEEPSEEK_API_KEY}
```

会在启动时被替换成真实环境变量值。

### 3.3 models 列表加载

当前 `Config.Models` 已经改成：

```go
[]*ModelConfig
```

配置格式对应：

```properties
models.0.provider=...
models.0.name=...
models.0.used=true
models.0.base_url=...
models.0.api_key=...
```

`loadModels(properties)` 会：

1. 遍历所有 `models.` 开头的键
2. 解析出 index 和 field
3. 先放进 `indexedModels map[int]*ModelConfig`
4. 最后按 index 排序，生成 `[]*ModelConfig`

### 3.4 选中当前模型

`ActiveModel()` 会遍历整个 `Models` 列表：

1. 按 `provider` 匹配
2. 按 `name` 匹配
3. 要求 `used=true`

如果：

- 找到了但 `used=false`，会报“配置存在但已禁用”
- 没找到启用项，会报“enabled model is not configured”

这一步是整个启动链路里很关键的一层校验。

---

## 4. HTTP 请求运行流程

主要看：

- `internal/httpapi/server.go`

### 4.1 创建 Session

`POST /v1/sessions`

执行：

```go
snapshot := s.sessions.Create(request.SessionID)
```

如果没传 `session_id`，系统会自动生成一个随机 ID。

会话存储当前是：

- 进程内内存
- 非持久化

所以服务重启后历史会丢失。

### 4.2 同步执行 Agent

`POST /v1/agent/runs`

流程：

1. 解析 `agent.RunRequest`
2. 创建超时上下文
3. 调用 `s.runtime.Run(ctx, request)`
4. 返回最终 JSON

当前这里的超时时间是：

```go
context.WithTimeout(r.Context(), 900*time.Second)
```

也就是 15 分钟。

### 4.3 流式执行

`POST /v1/agent/runs/stream`

流程：

1. 调用 `runtime.Stream(...)`
2. 设置 SSE Header
3. 先返回一个 `session` 事件
4. 再循环把模型的 delta 发回客户端

但当前有个硬限制：

```go
if len(r.tools.Definitions()) > 0 {
    return nil, "", errors.New("streaming with tool-enabled runtime is not available yet")
}
```

所以：

- **只要注册表里有工具**
- `/v1/agent/runs/stream` 当前就不可用

这意味着当前流式接口和 Tool Calling 闭环还没有打通。

---

## 5. Agent Runtime 主循环

位置：

- `internal/agent/runtime.go`

`Run()` 是整个 Agent 的核心。

### 5.1 校验输入并准备 Session

先检查：

```go
if strings.TrimSpace(request.Input) == ""
```

然后：

1. `Ensure(sessionID)` 确保 session 存在
2. 把当前用户输入追加到历史里

即：

```go
r.sessions.Append(snapshot.ID, llm.Message{Role:user, Content:request.Input})
```

### 5.2 进入 step loop

主循环：

```go
for step := 1; step <= r.maxSteps; step++
```

这就是 Runtime 的 agent loop。

每一轮都做：

1. 把历史消息 + prompt + tools 送给模型
2. 看模型返回的是最终答案，还是 tool call

### 5.3 调模型

`callModel(...)` 会构造：

```go
llm.Request{
    SystemPrompt: ...,
    Model: ...,
    Messages: history,
    Tools: toolDefinitions,
}
```

然后：

- 如果 `Tools` 非空 -> `llm.ToolCall(...)`
- 否则 -> `llm.Chat(...)`

这意味着 Runtime 的行为依赖于注册表里有没有工具。

### 5.4 模型直接结束

如果模型没有返回 `ToolCalls`：

```go
if len(modelResponse.ToolCalls) == 0
```

则视为本轮完成，直接返回：

- 最终消息
- 步数
- 本次 run 执行过的工具结果

### 5.5 模型要求调用工具

如果模型返回 tool call：

1. 先把 assistant 的 tool call 记录进 session
2. 调 `r.tools.Execute(...)`
3. 把工具 observation 再记录进 session
4. 进入下一轮 step，让模型基于 observation 继续推理

这里的 session 历史会形成标准的 ReAct/Tool Calling 轨迹：

```text
user -> assistant(tool call) -> tool(observation) -> assistant(final answer)
```

### 5.6 步数保护

如果一直循环都没有收敛，会报：

```go
agent exceeded max steps
```

它防止模型无限调用工具或者反复思考。

---

## 6. Session 管理流程

位置：

- `internal/session/manager.go`

当前是线程安全的内存会话管理器。

### 6.1 数据结构

每个 Session 保存：

- `ID`
- `Messages`
- `CreatedAt`
- `UpdatedAt`

### 6.2 核心方法

| 方法 | 作用 |
| --- | --- |
| `Create(id)` | 创建新会话 |
| `Ensure(id)` | 不存在就创建，存在就返回 |
| `Append(sessionID, messages...)` | 追加消息 |
| `Get(sessionID)` | 获取快照 |
| `GetOrEmpty(sessionID)` | 无则返回空消息集 |

### 6.3 当前特点

优点：

- 简单
- 并发安全
- 足够支撑最小 Agent Runtime

限制：

- 重启丢数据
- 没有 TTL
- 没有外部持久化
- 没有跨实例共享

---

## 7. Prompt 构造流程

位置：

- `internal/prompt/builder.go`

`SystemPrompt(tools []tool.Definition)` 会做两件事：

1. 写固定的 agent 行为约束
2. 把当前已注册工具列表拼进去

如果没有工具，会明确写：

```text
- no tools registered yet
```

所以 prompt 层天然感知工具注册状态。

这也是之前“模型说没有 tools 可用”的直接来源之一：当注册表为空时，prompt 本身就会告诉模型当前无工具。

---

## 8. Tool Registry 流程

位置：

- `internal/tool/registry.go`

### 8.1 注册

调用：

```go
registry.Register(definition, handler)
```

会把工具存到：

```go
map[string]registeredTool
```

工具名唯一，不允许重复注册。

### 8.2 暴露给模型

`LLMDefinitions()` 会把内部定义转成：

- `llm.ToolDefinition`

用于在模型请求里声明函数工具。

### 8.3 执行

模型返回 tool call 后，Runtime 最终会调用：

```go
registry.Execute(ctx, tool.Call{...})
```

按名称查找 handler，然后执行。

如果找不到，会返回：

```text
tool "xxx" is not registered
```

---

## 9. LLM Client 请求流程

位置：

- `internal/llm/openai/client.go`

虽然目录叫 `openai`，但实际已经是 **OpenAI-compatible chat completions client**。

### 9.1 请求地址

固定走：

```text
/chat/completions
```

即：

```go
const chatCompletionsPath = "/chat/completions"
```

所以当前兼容：

- OpenAI
- OpenRouter
- DeepSeek

前提是它们兼容这套请求格式。

### 9.2 构造 messages

`buildMessages(systemPrompt, history)` 会把内部 `llm.Message` 转成 chat completions 需要的格式。

特别是 assistant tool call 历史，会被转成：

- `assistant.tool_calls`
- 随后的 `tool` message

这一步很关键，因为它让多轮 Tool Calling 在下一轮调用里保持上下文连续。

### 9.3 构造 tools

如果 Runtime 传入了工具定义，会被转成：

```json
{
  "type": "function",
  "function": {
    "name": "...",
    "description": "...",
    "parameters": {...}
  }
}
```

并设置：

```json
"tool_choice": "auto"
```

这样模型就可以自主决定是否调用工具。

### 9.4 同步响应解析

`toLLMResponse(...)` 会从第一个 `choice` 中提取：

1. 普通文本内容
2. `tool_calls`

然后统一转回内部的：

- `llm.Response`

### 9.5 流式解析

`readStream(...)` + `parseEvent(...)` 负责解析 SSE。

当前只真正处理：

- 文本 delta
- finish reason
- `[DONE]`

也就是说当前流式路径对普通文本输出可用，但对 tool call 的流式事件没有做完整处理。

### 9.6 OpenRouter 特殊 Header

如果 Host 包含 `openrouter.ai`，会自动补：

- `HTTP-Referer`
- `X-Title`

这是为了兼容 OpenRouter 的一些接入要求。

---

## 10. 当前唯一可用业务工具：Market 概览

位置：

- `tools/market/register.go`

当前注册了：

- `market_token_overview`

### 10.1 输入

```json
{
  "token": "btc",
  "vs_currency": "usd"
}
```

### 10.2 处理流程

1. 反序列化参数
2. 根据入参选择数据源：
   - `token`：先查 CoinGecko `/search`，再查 `/coins/markets`
   - `token_address`：查 DexScreener `/latest/dex/tokens/{address}`
3. 解析返回值
4. 生成 observation 文本

### 10.3 返回给 Agent 的 observation

类似：

```text
Market overview for Bitcoin (BTC):
- asset_id: bitcoin
- price_usd: 118888.12
- market_cap_usd: 2300000000000.00
- 24h_volume_usd: 51000000000.00
- circulating_supply: 19700000.00
- total_supply: 21000000.00
- source: CoinGecko
```

模型再基于这个 observation 给最终答复。

---

## 11. 一次“查比特币行情”的完整运行时序

假设请求：

```json
{
  "input": "帮我查一下比特币现在的美元价格"
}
```

完整链路如下：

1. HTTP 收到 `POST /v1/agent/runs`
2. `handleRun()` 解析请求
3. `runtime.Run()` 追加 user message 到 session
4. `callModel()` 把：
   - system prompt
   - 历史消息
   - `market_token_overview` 工具定义
   发给模型
5. 模型返回：
   - `tool_calls: [{name:"market_token_overview", arguments:{"token":"btc","vs_currency":"usd"}}]`
6. Runtime 把 tool call 记入 session
7. Runtime 执行 `toolRegistry.Execute(...)`
8. Market 工具调用 CoinGecko，得到市场信息
9. Runtime 把 observation 写回 session
10. 进入下一轮 step，再调模型
11. 模型读取 observation，生成自然语言结果
12. Runtime 返回最终 `RunResponse`

这就是当前工程最典型、最完整的一条 Agent 闭环。

---

## 12. 当前工程的几个关键限制

### 12.1 大部分工具还是空的

虽然目录很多，但当前真正有实现的几乎只有：

- `market_token_overview`

所以工程骨架已经成型，但 Web3 能力远未完整。

### 12.2 Streaming 与 Tool Calling 没打通

当前一旦注册了工具：

- `/v1/agent/runs/stream` 就不可用

这是现在最明显的一处运行能力缺口。

### 12.3 Session 只在内存中

当前 session 不能：

- 持久化
- 跨实例共享
- 重启恢复

### 12.4 MCP 还未接入实际主流程

`mcp.Registry` 已创建，但目前只是占位。

### 12.5 HTTP API 的错误语义还比较粗

现在大部分错误都直接返回：

- `400 Bad Request`

后续更合理的拆法通常是：

- 400：输入问题
- 500：内部错误
- 502/503：上游 LLM 或外部工具问题

---

## 13. 总结

当前工程已经具备一个真实的最小 Agent Runtime 主干：

1. **启动阶段**
   - 读配置
   - 选模型
   - 初始化 session / prompt / tools / llm / runtime / http server

2. **运行阶段**
   - HTTP 接入请求
   - Session 记录消息
   - Prompt 注入工具信息
   - LLM 决定是否调用工具
   - Tool Registry 执行工具
   - observation 回灌模型
   - 返回最终结果

它当前最像一个“可演进的 Runtime 骨架”，而不是功能完整的 Web3 Agent。主干已经成立，后续复杂度主要会增长在：

- 更多 Web3 tools
- Planner / Workflow
- 持久化 memory / session
- 流式 tool calling
- MCP 接入
