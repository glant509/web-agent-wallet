# web3-service-agent

`web3-service-agent` 是一个面向 Web3 场景的 Go Agent Runtime 骨架。当前版本先把长期可迭代的主干搭起来：HTTP 入口、Session 管理、Prompt Builder、LLM 抽象、Tool Registry、Agent Loop、Provider 可配置的模型接入层、Streaming 预留、MCP 预留、以及按能力分类的工具插件目录。

## 当前目录结构

```text
cmd/web3-service-agent/     # 服务入口
internal/agent/             # Agent Runtime 主循环
internal/config/            # 环境配置
internal/httpapi/           # HTTP / SSE 接口
internal/llm/               # LLM 抽象
internal/llm/openai/        # OpenAI-compatible Chat Completions 适配
internal/mcp/               # MCP 接入预留
internal/prompt/            # Prompt Builder
internal/session/           # Session Manager
internal/tool/              # Tool Registry / Executor
tools/                      # Web3 Tool 插件入口
```

## 已有能力

- `POST /v1/sessions`：创建会话
- `POST /v1/agent/runs`：执行一次完整 Agent Run
- `POST /v1/agent/runs/stream`：SSE 流式输出
- `GET /healthz`：健康检查
- `GET /`：移动端 H5 入口，内置聊天 / 行情 / 交易 / 广场 / 资产五个底部菜单页
- Agent Runtime 已具备：
  - 多轮 Session 历史
  - Tool Calling 主循环
  - Tool Registry 插件式注册
  - Provider 可配置的 LLM 接入层
  - MCP Registry 预留

## 运行方式

```bash
export OPENAI_API_KEY=your_key
go run ./cmd/web3-service-agent
```

默认监听 `:8080`。

项目默认从根目录的 `etc/application.properties` 读取配置。

## 核心配置文件

```properties
service.name=web3-service-agent
service.port=8080
service.agentMaxSteps=8
service.modelProvider=deepseek
service.modelName=deepseek-chat

models.0.provider=openrouter
models.0.name=openai/gpt-4.1-mini
models.0.used=true
models.0.base_url=https://openrouter.ai/api/v1
models.0.api_key=${OPENROUTER_API_KEY}

models.1.provider=deepseek
models.1.name=deepseek-chat
models.1.used=true
models.1.base_url=https://api.deepseek.com
models.1.api_key=${DEEPSEEK_API_KEY}

marketProviders.0.used=true
marketProviders.0.name=coingecko
marketProviders.0.base_url=https://api.coingecko.com/api/v3
marketProviders.0.api_key=${CG-1HoLG61sqiEsjoXaieJUwmkq}
```

启动时会遍历 `models.*` 列表，根据 `service.modelProvider` 和 `service.modelName` 选出当前模型，并且只会选中 `used=true` 的项。

行情能力会遍历 `marketProviders.*` 列表，并使用 **第一个 `used=true` 的 provider** 作为当前市场数据源。当前已抽象 provider 层，不同数据源后续可以接入不同请求路径；现阶段默认实现是 CoinGecko。

`models.<index>.api_key` 和 `marketProviders.<index>.api_key` 都支持 `${ENV_NAME}` 形式的环境变量展开，便于把密钥留在环境变量里，不直接写入仓库。

如果要切到 DeepSeek，可以把配置改成：

```properties
service.modelProvider=deepseek
service.modelName=deepseek-chat

models.0.provider=deepseek
models.0.name=deepseek-chat
models.0.used=true
models.0.base_url=https://api.deepseek.com
models.0.api_key=${DEEPSEEK_API_KEY}
```

## 关键配置项

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `service.name` | `web3-service-agent` | 服务名称 |
| `service.port` | `8080` | HTTP 监听端口 |
| `service.agentMaxSteps` | `8` | 单次 Agent Loop 最大步数 |
| `service.modelProvider` | `openrouter` / `deepseek` | 当前启用的模型提供商 |
| `service.modelName` | `openai/gpt-4.1-mini` | 当前启用的模型名称 |
| `models.<index>.provider` | - | 第 N 个模型的 provider |
| `models.<index>.name` | - | 第 N 个模型的名称 |
| `models.<index>.used` | `true/false` | 第 N 个模型是否可被选中 |
| `models.<index>.base_url` | 视 provider 而定 | 第 N 个模型的 API Base URL |
| `models.<index>.api_key` | - | 第 N 个模型的 API Key |
| `marketProviders.<index>.used` | `true/false` | 第 N 个行情源是否启用 |
| `marketProviders.<index>.name` | - | 第 N 个行情源名称 |
| `marketProviders.<index>.base_url` | 视 provider 而定 | 第 N 个行情源 API Base URL |
| `marketProviders.<index>.api_key` | - | 第 N 个行情源 API Key |

## 设计取向

这个版本刻意不放 Demo 业务逻辑，只放真正会长期保留的工程主干：

1. Agent Runtime 与具体链能力解耦
2. LLM Provider 与 Runtime 解耦
3. Tool 采用插件注册，新增链能力不改 Runtime
4. MCP 提前预留，后续 Wallet / DEX / Browser / GitHub 都能接进来

## 工具分类约定

`tools/` 优先按能力分类，而不是按单条链拆目录。当前建议主分类是：

- `market`：价格、总量、流动性、市值等市场信息
- `bridge`：跨链桥相关能力
- `dex`：交易、询价、路由相关能力
- `yield`：收益、质押、借贷相关能力

例如现在已经把原来的 `btc_price_lookup` 收敛成了通用的市场工具：

- 查主流资产：传 `token=BTC`、`token=ETH`、`token=SOL`
- 查链上 token：传 `token_address=<address>`，必要时再带 `chain`
- 基础行情：用 `market_token_overview`
- K 线 / 历史 K 线：用 `market_token_klines`，最近行情可传 `days`，历史区间可传 `from_timestamp` + `to_timestamp`
- H5 market 页会把 `market_token_klines` 返回的 candle 数据自动渲染成类似 DEX / wallet 的可视化蜡烛图卡片，而不只是纯文本
- H5 market 页在切换进入时，还会自动拉取并展示 **市值前 10** 代币，包含名称、当前价格和 24h 涨跌幅
- 点击 market 榜单里的代币后，会切换到 **trade** 页，并展示类似 OKX / Bitget DEX 风格的交易数据看板；同时新增了 `dex_token_trade_dashboard` tool

这样后续新增链和 token 时，不需要再为每条链或每个币单独新增一个查询基础行情的工具。

下一步最适合补的是第一批真实 Web3 Tool，例如 `tools/dex/quote`、`tools/bridge/route`、`tools/yield/opportunity`。
