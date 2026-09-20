# Agent Wallet 多端产品架构与实施方案

## 1. 文档目的

当前 `web3-service-agent` 已基本具备 Web 端 Agent Wallet 的主体框架，主要包括：

- Agent Runtime、Session、Prompt、LLM Provider 和 Tool Calling；
- Web3 行情、资产余额、交易历史等查询能力；
- 多链钱包创建、导入、BIP39/BIP44 派生和账户管理；
- Send、Receive、手续费预估、交易详情确认；
- 客户端本地签名、服务端广播；
- 钱包保险库、自动锁定、助记词和私钥导出；
- Trace、Span、结构化日志和 HTTP 请求日志。

后续目标是将当前 Web 服务逐步扩展为：

- iOS App；
- Android App；
- Chrome 浏览器插件；
- 保留现有 Web 版本；
- 多端尽可能共享钱包能力、API Client 和 UI 组件。

本文件说明推荐架构、不同实施方案、安全边界、阶段计划，以及未来 DApp、跨链等功能的扩展方式。

---

## 2. 当前系统的判断

当前项目已经形成了两个基本部分。

### 2.1 Go 服务端

服务端目前负责：

- Agent Runtime；
- LLM 调用；
- Tool Registry；
- 行情供应商调用；
- 链 RPC 调用；
- 资产、历史和交易准备接口；
- 已签名交易广播；
- 日志和链路追踪。

主要代码位于：

```text
cmd/web3-service-agent/
internal/agent/
internal/httpapi/
internal/llm/
internal/marketdata/
internal/session/
internal/tool/
tools/
```

### 2.2 Web 钱包客户端

当前 Web 页面负责：

- 钱包创建和导入；
- 助记词生成；
- BIP39 Seed 生成；
- BIP44 账户派生；
- 钱包加密保险库；
- 本地交易签名；
- UI 状态和交互；
- 调用 Go HTTP API。

主要代码位于：

```text
internal/httpapi/ui/index.html
internal/httpapi/ui/wallet_derivation.js
internal/httpapi/ui/evm_signer.js
```

当前设计中，服务端准备交易，浏览器本地签名，然后将签名结果交给服务端广播。这一安全边界是正确方向，应当在移动端和插件端继续保留。

---

## 3. 推荐的目标架构

不建议把整个 Go 服务直接嵌入 iOS、Android 或浏览器插件。推荐拆分为三层：

```text
┌──────────────────────────────────────────────┐
│                  多端客户端                   │
│                                              │
│  Web       iOS       Android       Chrome    │
│   │         │           │             │      │
│   └─────────┴───────────┴─────────────┘      │
│                   │                          │
│        Wallet Core / API Client / UI         │
└───────────────────┬──────────────────────────┘
                    │ HTTPS / SSE / WebSocket
┌───────────────────▼──────────────────────────┐
│             Go Agent Wallet 服务             │
│                                              │
│ Agent / LLM / Tool / Market / RPC / History  │
│ Transaction Prepare / Simulation / Broadcast │
│ Auth / Session / Trace / Risk Control        │
└──────────────────────────────────────────────┘
```

### 3.1 云端 Go 服务的职责

以下能力应继续放在服务端：

- Agent Runtime 和模型调用；
- 模型、行情和第三方服务 API Key；
- 行情聚合；
- RPC 节点配置、故障切换和限流；
- 资产余额和历史查询；
- Gas、手续费和交易参数预估；
- 未签名交易构建；
- 交易模拟和风险检测；
- 已签名交易广播；
- Agent 会话和非敏感用户配置同步；
- 服务端日志、Trace、Span、告警和审计。

### 3.2 客户端钱包内核的职责

以下能力必须留在用户设备：

- 助记词生成和导入；
- BIP39 Seed 计算；
- BIP44 账户派生；
- 私钥派生；
- 交易签名；
- 消息签名和 Typed Data 签名；
- 钱包保险库加密和解密；
- 密码、生物识别和自动锁定；
- 导出助记词或私钥；
- 交易确认页面和最终用户授权。

### 3.3 服务端永远不能接收的数据

以下内容不能上传服务端，也不能写入日志、埋点或崩溃报告：

- 钱包密码；
- 助记词；
- 明文 Seed；
- 私钥；
- 解密后的钱包保险库；
- 生物识别数据；
- 任何能够恢复钱包的材料。

---

## 4. 三种实现方案

## 4.1 方案 A：快速 Hybrid 封装

使用 Capacitor 将现有 HTML、CSS、JavaScript 页面封装成 iOS 和 Android App。

Capacitor 可以嵌入现有 Web 工程，并通过 Swift、Java 或 Kotlin Plugin 接入原生功能：

- 官方文档：https://capacitorjs.com/docs

### 优点

- 当前页面复用率最高；
- iOS 和 Android 共用一套 UI；
- 最快形成可安装的移动端版本；
- 可以接入相机、二维码、剪贴板、生物识别和推送。

### 缺点

- 当前 `index.html` 体积过大，需要先拆分；
- 钱包敏感数据仍然会进入 JavaScript 内存；
- 原生交互体验弱于 SwiftUI 或 Jetpack Compose；
- 如果只做 WebView 包装而不做安全存储改造，不适合正式钱包产品。

### 适用阶段

- 内部演示；
- TestFlight；
- Android 内测；
- 产品流程验证；
- 不承载大额资产的早期版本。

## 4.2 方案 B：共享核心、多端外壳

这是当前最推荐的方案。

建议逐步整理为：

```text
apps/
  web/
  mobile/
  extension/

packages/
  wallet-core/
  api-client/
  ui-components/
  design-tokens/

server/
  Go 服务端代码
```

其中：

- Web 继续运行现有网站；
- Mobile 使用 Capacitor 生成 iOS 和 Android 工程；
- Chrome 使用 Manifest V3 单独构建；
- 各端共用 Wallet Core、API Client 和大部分 UI 组件；
- iOS 和 Android 使用独立的原生安全存储适配器；
- Chrome 使用扩展专用的存储和后台 Service Worker。

### 优点

- 开发效率和安全性平衡较好；
- 不需要立即重写全部页面；
- Web、移动端和插件可以共享大部分业务代码；
- 后期可以逐步替换钱包核心实现；
- 适合当前工程的技术基础。

### 缺点

- 需要先完成前端模块化；
- 需要维护不同平台的安全存储适配器；
- 浏览器插件仍有单独的生命周期和安全模型。

## 4.3 方案 C：原生客户端和 Rust Wallet Core

长期可以将钱包核心实现为 Rust，并编译成：

```text
Rust Wallet Core
  ├── iOS XCFramework
  ├── Android JNI/AAR
  └── Chrome/Web WASM
```

客户端 UI 可以使用：

- iOS：SwiftUI；
- Android：Jetpack Compose；
- Chrome：React/TypeScript；
- Web：React/TypeScript；
- 服务端：继续使用当前 Go 服务。

### 优点

- 多端共享同一套密码学和签名逻辑；
- 类型、内存和边界更加可控；
- 方便做测试向量、模糊测试和安全审计；
- 更适合长期承载大额资产。

### 缺点

- 开发成本最高；
- 需要额外掌握 Rust、FFI、JNI、XCFramework 和 WASM；
- UI 可能需要大量重写；
- 发布速度明显慢于 Hybrid 方案。

### 建议

现阶段不需要直接进入完整方案 C。可以先按照统一 Wallet Core 接口开发 TypeScript/JavaScript 版本，等产品验证后再替换为 Rust 实现。

---

## 5. 最终推荐方案

推荐选择方案 B，并为未来方案 C 预留接口。

推荐组合：

```text
服务端：Go
共享 Wallet Core：第一阶段 TypeScript，成熟后可迁移 Rust
共享 API Client：TypeScript
共享 UI：React 或其他组件化 Web 框架
iOS/Android：Capacitor + 原生安全插件
Chrome：Manifest V3 + 独立 Extension Shell
```

当前页面是移动端布局，所以先使用 Capacitor 做移动端的投入产出比最高。React Native、Flutter、SwiftUI 和 Compose 都需要明显更多的 UI 重写，不适合作为当前第一步。

---

## 6. 前端模块化改造

当前 `index.html` 包含大量样式、页面结构和业务逻辑。多端开发前应先拆分。

推荐模块：

```text
wallet-core/
  mnemonic.ts
  seed.ts
  bip44.ts
  account.ts
  address.ts
  vault.ts
  signer.ts
  transaction.ts

api-client/
  http.ts
  trace.ts
  agent.ts
  assets.ts
  history.ts
  transactions.ts

stores/
  wallet-store.ts
  account-store.ts
  asset-store.ts
  history-store.ts
  agent-store.ts

components/
  WalletUnlock
  AccountSelector
  AssetList
  SendForm
  ReceivePage
  TransactionReview
  TransactionHistory
  AgentChat
```

### 必须建立的抽象接口

#### SecureVault

```text
loadEncryptedVault()
saveEncryptedVault()
deleteEncryptedVault()
unlock()
lock()
isUnlocked()
```

不同平台分别实现：

```text
WebSecureVault       -> IndexedDB
IOSSecureVault       -> Keychain + App 私有存储
AndroidSecureVault   -> Android Keystore + App 私有存储
ChromeSecureVault    -> chrome.storage.local/session
```

#### WalletCore

```text
createMnemonic()
importMnemonic()
deriveAccount()
deriveAddress()
derivePrivateKey()
signTransaction()
signMessage()
signTypedData()
clearSensitiveMemory()
```

#### AgentWalletAPI

```text
createSession()
runAgent()
loadPortfolio()
loadHistory()
prepareTransaction()
simulateTransaction()
broadcastTransaction()
```

---

## 7. iOS 实现方案

## 7.1 工程形态

第一阶段使用 Capacitor 创建 iOS 工程：

```text
Web UI
  ↓
Capacitor Bridge
  ↓
Swift SecureWallet Plugin
  ↓
Keychain / LocalAuthentication / App Storage
```

## 7.2 钱包保险库

推荐流程：

1. 用户创建或导入助记词；
2. 客户端生成随机的 256 位保险库加密密钥；
3. 使用 AES-256-GCM 加密助记词、Keyring 和账户配置；
4. 加密后的保险库保存在 App 私有存储；
5. 保险库加密密钥存入 Keychain；
6. Keychain 项目要求 Face ID、Touch ID 或设备密码授权；
7. 解锁后只在内存中短暂保留 Seed；
8. 自动锁定、退到后台或设备锁屏时清理敏感内存。

Apple Keychain 适合保存密码、密钥等小型敏感数据：

- https://developer.apple.com/documentation/security/keychain-services
- https://developer.apple.com/documentation/security/storing-keys-in-the-keychain

生物识别应使用 LocalAuthentication，并用于保护 Keychain 项目的访问：

- https://developer.apple.com/documentation/localauthentication

## 7.3 Secure Enclave 注意事项

不能简单认为所有区块链私钥都可以直接保存到 Secure Enclave 并完成签名。

例如 EVM 账户使用 `secp256k1`，而 Secure Enclave 原生支持的算法范围并不等同于钱包所需的全部算法。因此比较现实的方案是：

- Secure Enclave/Keychain 保护保险库加密密钥；
- 助记词和链私钥以密文形式保存；
- 用户通过 Face ID 或 Touch ID 授权；
- 在受控钱包内核中短暂解密并签名；
- 签名完成后清除敏感内存。

## 7.4 iOS 必要功能

- Face ID/Touch ID 解锁；
- App 进入后台立即隐藏敏感页面；
- App Switcher 快照遮罩；
- 助记词和私钥页面禁止截图或给出高风险提示；
- Universal Link/Deep Link；
- 二维码扫描；
- 推送通知；
- 网络权限和证书策略；
- TestFlight、崩溃报告脱敏；
- App Store 隐私声明。

---

## 8. Android 实现方案

## 8.1 工程形态

第一阶段使用 Capacitor 创建 Android 工程：

```text
Web UI
  ↓
Capacitor Bridge
  ↓
Kotlin SecureWallet Plugin
  ↓
Android Keystore / BiometricPrompt / App Storage
```

## 8.2 钱包保险库

推荐流程与 iOS 一致，但使用 Android Keystore：

1. 生成保险库加密密钥；
2. 优先使用硬件支持的 Keystore；
3. 支持时使用 TEE 或 StrongBox；
4. 使用 BiometricPrompt 或设备密码授权；
5. 保险库密文放在 App 私有存储；
6. 自动锁定后清除 Seed 和派生私钥；
7. 禁止未加密钱包数据进入系统备份。

Android Keystore 官方文档：

- https://developer.android.com/privacy-and-security/keystore

## 8.3 Android 必要功能

- 指纹、人脸或设备密码解锁；
- `FLAG_SECURE` 保护敏感页面；
- 后台任务预览遮罩；
- App 私有存储；
- Android App Link；
- 相机扫码；
- Play Integrity 风险信号；
- Root、Hook 和篡改风险检测；
- Google Play Data Safety 声明；
- 崩溃日志和埋点脱敏。

---

## 9. Chrome 浏览器插件实现方案

## 9.1 第一阶段：仅做钱包快捷入口

当前尚未计划接入第三方 DApp，因此第一阶段的 Chrome 插件可以只提供：

- 钱包创建、导入和解锁；
- 账户切换；
- 资产列表；
- Send、Receive；
- History；
- Agent 对话；
- 网络切换；
- 设置和导出密钥。

这一阶段不向网页注入 `window.ethereum`，第三方网站不能调用钱包，攻击面更小。

## 9.2 Manifest V3 结构

推荐结构：

```text
extension/
  manifest.json
  service-worker.ts
  popup.html
  popup.ts
  sidepanel.html
  sidepanel.ts
  approval.html
  approval.ts
  storage.ts
  wallet-bridge.ts
```

各模块职责：

- Popup：简单账户、余额和快捷操作；
- Side Panel：完整钱包和 Agent 页面；
- Approval Window：未来的连接、签名和交易确认；
- Service Worker：钱包状态、权限、消息路由；
- `chrome.storage.local`：保存加密保险库；
- `chrome.storage.session`：保存临时解锁状态；
- HTTPS API：调用 Go 服务端。

Manifest V3 使用短生命周期 Service Worker，不能依赖全局变量长期保存状态：

- https://developer.chrome.com/docs/extensions/develop/migrate/to-service-workers

Chrome 推荐使用 Extension Storage API，而不是继续依赖普通 `localStorage`：

- https://developer.chrome.com/docs/extensions/reference/api/storage

## 9.3 Chrome 安全要求

- 所有可执行 JavaScript、WASM 和依赖必须随插件打包；
- 不加载或执行远程代码；
- 不使用 `eval`、`new Function` 或动态脚本；
- Content Security Policy 使用最严格规则；
- 只申请必要权限；
- Service Worker 中处理敏感权限；
- Content Script 的消息一律视为不可信；
- 不把 Seed、私钥或助记词发送给网页；
- 不把敏感数据放进 `chrome.storage.sync`；
- 插件自动锁定；
- 浏览器重启后默认保持锁定。

相关文档：

- https://developer.chrome.com/docs/extensions/develop/migrate/improve-security
- https://developer.chrome.com/docs/extensions/develop/security-privacy/stay-secure

---

## 10. EIP-1193、EIP-6963 和 DApp 授权

## 10.1 当前是否需要实现

当前钱包暂时没有接入第三方 DApp，因此现阶段不需要实现：

- EIP-1193 Provider；
- EIP-6963 钱包发现；
- DApp 网站授权管理；
- Content Script 和 In-page Provider；
- WalletConnect 类连接能力。

第一阶段 Chrome 插件只作为钱包快捷入口即可。

## 10.2 EIP-1193 服务的功能

EIP-1193 是 EVM DApp 调用浏览器钱包的 Provider 标准，主要支持：

- 请求连接钱包；
- 获取授权账户；
- 查询当前网络；
- 请求切换网络；
- 请求签名消息；
- 请求签名 Typed Data；
- 请求发送交易；
- 监听账户和网络变化。

典型方法包括：

```text
eth_requestAccounts
eth_accounts
eth_chainId
wallet_switchEthereumChain
personal_sign
eth_signTypedData_v4
eth_sendTransaction
```

官方规范：

- https://eips.ethereum.org/EIPS/eip-1193

## 10.3 EIP-6963 服务的功能

当用户同时安装 Agent Wallet、MetaMask、OKX Wallet 等多个钱包时，EIP-6963 允许 DApp 发现并展示多个 Provider，避免所有钱包争抢 `window.ethereum`。

它主要用于：

- 在 DApp 钱包选择列表中显示 Agent Wallet；
- 提供钱包名称、图标和唯一标识；
- 避免多个浏览器钱包相互覆盖；
- 让用户明确选择使用哪个钱包。

官方规范：

- https://eips.ethereum.org/EIPS/eip-6963

## 10.4 DApp 授权服务的功能

DApp 授权用于控制：

- 哪个网站可以查看钱包地址；
- 网站可以使用哪个账户；
- 网站可以访问哪些网络；
- 用户何时授权、断开和重新连接；
- 网站发起签名或交易时如何确认；
- 哪些权限可以长期保存。

连接授权和交易授权必须分开：

- 用户同意连接，只表示允许网站看到指定地址；
- 每一次消息签名、Typed Data 签名和交易发送仍要单独确认；
- 高风险签名不能因为网站已连接而自动批准。

## 10.5 未来接入 DApp 时的插件结构

```text
DApp 页面
   │ EIP-1193 / EIP-6963
   ▼
inpage-provider.ts
   │ window.postMessage
   ▼
content-script.ts
   │ chrome.runtime.sendMessage
   ▼
background service worker
   │
   ├── 权限检查
   ├── Origin 校验
   ├── Chain 校验
   ├── 账户授权
   └── 打开 Approval Window
             │
             ▼
        用户确认并本地签名
```

只有在计划让用户从 Uniswap、OpenSea、Aave 等网站选择并连接 Agent Wallet 时，才需要进入这一阶段。

---

## 11. DApp 接入与跨链的区别

EIP-1193、EIP-6963 和 DApp 授权与跨链没有直接关系。

| 能力 | 解决的问题 |
| --- | --- |
| EIP-1193 | DApp 如何请求 EVM 钱包执行操作 |
| EIP-6963 | DApp 如何发现多个浏览器钱包 |
| DApp 授权 | 哪个网站可以访问哪个账户 |
| Swap | 如何在一条链内兑换 Token |
| Bridge | 如何把资产从源链转移到目标链 |

跨链需要另外实现：

- Bridge Provider 接入；
- 多路线报价；
- 源链和目标链状态；
- 源链授权和锁定交易；
- 目标链铸造或释放；
- 跨链消息跟踪；
- 超时、失败和退款；
- 滑点和最小到账金额；
- Bridge 合约和第三方风险提示。

实现 EIP-1193 不会自动获得跨链能力；实现 Bridge 也不要求一定先接入 DApp。

---

## 12. 服务端需要补充的能力

多端正式上线前，服务端建议增加：

### 12.1 用户和设备认证

- 用户账户或匿名设备身份；
- Access Token 和 Refresh Token；
- 设备列表；
- 设备撤销；
- 风险登录检测；
- 请求签名或设备证明；
- API 限流。

用户身份和区块链地址不应被强制绑定。应允许用户不注册账户也能创建本地钱包，再选择是否开启云端同步。

### 12.2 Session 持久化

当前 Agent Session 不应长期只保存在服务进程内存中，应逐步迁移为：

- Redis：短期 Session 和流式运行状态；
- PostgreSQL：用户、设备、会话索引和非敏感配置；
- 对话记录可由用户选择是否保存；
- 所有敏感字段脱敏。

### 12.3 RPC 和广播

- 多 RPC Provider；
- 超时和重试；
- 熔断；
- 链 ID 校验；
- 广播结果校验；
- Transaction Hash 一致性校验；
- 防止重复广播；
- 交易状态跟踪；
- Reorg 处理。

### 12.4 交易模拟和风险提示

服务端构建或接收交易后，应返回：

- 原生 Token 变化；
- ERC-20 Token 变化；
- NFT 变化；
- Token Approval 数量；
- 无限授权提示；
- 合约地址和验证状态；
- 风险地址提示；
- 预计 Gas；
- 交易失败原因；
- 可读交易摘要。

客户端必须根据这些数据生成确认页面，不能只展示十六进制 calldata。

---

## 13. 统一交易流程

推荐所有平台使用相同流程：

```text
1. 用户发起 Send/Swap/Bridge/DApp 交易
2. 客户端收集用户输入
3. 服务端准备未签名交易
4. 服务端模拟交易并进行风险分析
5. 客户端校验服务端返回的链、地址、金额和 calldata
6. 客户端展示完整交易确认页面
7. 用户通过密码或生物识别授权
8. Wallet Core 本地签名
9. 客户端清理私钥和临时签名数据
10. 服务端广播已签名交易
11. 客户端显示 Transaction Hash
12. 后台跟踪 pending、confirmed、failed 或 replaced
```

### 客户端必须二次校验

客户端不能盲目信任服务端准备的交易，至少校验：

- `chain_id`；
- `from` 地址；
- `to` 地址；
- `value`；
- Token 合约；
- Token 数量；
- Gas 参数；
- Nonce；
- calldata 对应的方法和参数。

这样即使服务端受到攻击，也不能静默替换收款地址或 Token 数量。

---

## 14. Wallet Core 测试和审计

Wallet Core 应独立于 UI 进行测试。

### 必要测试

- BIP39 官方测试向量；
- BIP32/BIP44 派生测试向量；
- 不同账户编号测试；
- EVM 地址派生；
- EVM Legacy/EIP-1559 交易签名；
- ERC-20 Transfer 编码；
- Typed Data 签名；
- 错误 Chain ID；
- 错误 Nonce；
- 极大金额和小数精度；
- 钱包加密、解密和密码错误；
- 自动锁定；
- 敏感数据清理；
- 多平台相同输入产生相同地址和签名。

### 安全工作

- 依赖锁定和供应链扫描；
- 静态分析；
- 模糊测试；
- 密码学实现不自行发明算法；
- 发布包可复现构建；
- 钱包核心和插件进行第三方安全审计；
- 上线前建立漏洞响应和版本吊销机制。

---

## 15. 推荐实施阶段

## 阶段 0：当前 Web 版本收口

目标：把现有功能稳定下来，为多端改造建立基线。

- [ ] 完善全部接口测试；
- [ ] 固化 Wallet Core 测试向量；
- [ ] 完善交易详情校验；
- [ ] 完善错误码；
- [ ] 完善 Trace 和 Span；
- [ ] 清理日志中的敏感字段；
- [ ] 完成依赖清单；
- [ ] 明确支持的链和 Token。

## 阶段 1：前端模块化

目标：把当前单体页面拆成可复用工程。

- [x] 引入 TypeScript 和严格类型检查；
- [x] 拆分 Wallet Core（保险库加解密、KDF、BIP39 seed、编码和随机数）；
- [x] 拆分 API Client（统一 URL、Trace/Span 和 JSON 错误）；
- [x] 拆分基础状态管理，并抽取账户、历史交易、交易构建和行情领域逻辑；
- [x] 建立 UI Components 基础包（地址、金额、可见性和 busy 状态）；
- [ ] 建立统一 Design Token；
- [x] 建立 Web、Mobile、Extension 统一 TypeScript 构建流程；
- [x] 保持现有 Go Embed 能继续发布 Web 页面。

## 阶段 2：iOS 和 Android MVP

目标：使用 Capacitor 形成可安装版本。

- [ ] 创建 Capacitor 工程；
- [ ] 接入现有 UI；
- [ ] iOS Keychain Plugin；
- [ ] Android Keystore Plugin；
- [x] 接入 Face ID/Touch ID/BiometricPrompt（真机交互验收待完成）；
- [ ] 后台自动锁定；
- [ ] 敏感页面遮罩；
- [ ] 相机扫码；
- [ ] Deep Link/App Link；
- [ ] TestFlight 和 Android 内测。

## 阶段 3：Chrome 钱包快捷入口

目标：先提供独立钱包能力，不接第三方 DApp。

- [ ] Manifest V3；
- [ ] Popup；
- [ ] Side Panel；
- [ ] Service Worker；
- [ ] Chrome SecureVault；
- [ ] 自动锁定；
- [ ] API Client；
- [ ] Chrome Web Store 隐私和权限说明；
- [ ] 不注入 `window.ethereum`。

## 阶段 4：多端账号和云服务

目标：实现 Agent 和非敏感配置同步。

- [ ] 用户和设备认证；
- [ ] Session 持久化；
- [ ] 多设备管理；
- [ ] 设备撤销；
- [ ] 对话历史可选同步；
- [ ] 地址簿可选加密同步；
- [ ] 推送通知；
- [ ] 交易状态同步。

## 阶段 5：DApp 接入（按需）

只有产品决定接入第三方 DApp 时再实施：

- [ ] EIP-1193 Provider；
- [ ] EIP-6963 Discovery；
- [ ] Content Script；
- [ ] In-page Provider；
- [ ] DApp Origin 授权；
- [ ] 账户和网络权限；
- [ ] 签名确认页面；
- [ ] Typed Data 可读解析；
- [ ] 恶意请求防护；
- [ ] DApp 连接管理页面。

## 阶段 6：Swap 和 Bridge（按需）

- [ ] DEX Quote；
- [ ] Swap Route；
- [ ] Token Approval；
- [ ] 滑点控制；
- [ ] Bridge Provider；
- [ ] 跨链状态机；
- [ ] 风险提示；
- [ ] 失败恢复和退款；
- [ ] 多供应商比较。

## 阶段 7：Wallet Core Rust 化（长期）

- [ ] 定义稳定 Wallet Core API；
- [ ] Rust 实现和测试向量；
- [ ] iOS FFI；
- [ ] Android JNI；
- [ ] WASM；
- [ ] 多端一致性测试；
- [ ] 第三方安全审计。

---

## 16. 当前优先级建议

结合当前产品状态，优先级建议如下：

1. 完善钱包保险库和本地签名安全；
2. 完善多链余额、历史和发送交易；
3. 增加交易模拟、Token Approval 和风险提示；
4. 将当前 Web 页面组件化；
5. 抽取 Wallet Core 和 API Client；
6. 使用 Capacitor 实现 iOS/Android MVP；
7. 实现 Chrome 钱包快捷入口；
8. 增加正式用户认证和 Session 持久化；
9. 产品确定接入 DApp 后再实现 EIP-1193/EIP-6963；
10. 产品确定跨链需求后再实现 Bridge；
11. 用户量和资金规模增长后迁移 Rust Wallet Core。

---

## 17. 不推荐的做法

- 不要把助记词或私钥上传到 Go 服务端；
- 不要把模型和第三方 API Key 打包进 App；
- 不要把整个 Go HTTP 服务直接嵌入移动端作为主要方案；
- 不要在四个平台分别重写四套钱包派生和签名逻辑；
- 不要只用普通 `localStorage` 保存钱包保险库；
- 不要把解锁密码保存到 Keychain、Keystore 或浏览器存储；
- 不要允许 DApp 静默签名或静默发送交易；
- 不要在日志、Trace、埋点、Sentry 或崩溃报告中记录敏感信息；
- 不要从 CDN 或远程服务器加载 Chrome 插件可执行代码；
- 不要把“连接钱包”与“授权交易”等同；
- 不要把 EIP-1193 当成跨链方案。

---

## 18. 结论

当前最适合 Agent Wallet 的路线是：

```text
保留 Go 云端服务
        +
抽取共享 Wallet Core 和 API Client
        +
Capacitor 实现 iOS/Android
        +
Manifest V3 实现 Chrome 钱包入口
        +
未来按需增加 DApp Provider、Swap 和 Bridge
```

这样做能够：

- 最大限度复用当前代码；
- 保留正确的本地签名安全边界；
- 避免维护四套业务逻辑；
- 快速形成 iOS、Android 和 Chrome 产品；
- 为未来 Rust Wallet Core 和原生 UI 留出升级空间；
- 在当前没有 DApp 和跨链需求时控制开发范围，避免过早增加复杂度。

---

## 19. 本地浏览器开发和验证

完成 Web、iOS、Android 和 Chrome 改造后，必须继续保留本地浏览器验证能力。共享 Wallet Core、API Client 和 UI 组件不能依赖某一个原生平台才能运行。

推荐保留四个独立运行目标：

```text
共享 Wallet Core + API Client + UI Components
                 │
       ┌─────────┼──────────┬──────────┐
       ▼         ▼          ▼          ▼
     Web      iOS App   Android App  Chrome 插件
```

### 19.1 Web 本地运行方式

现有 Go 服务内嵌 Web 页面应继续保留。开发者可以启动：

```bash
go run ./cmd/web3-service-agent
```

然后访问：

```text
http://localhost:8081
```

Web 环境继续使用：

- IndexedDB 保存加密后的钱包保险库；
- Web Crypto API 进行保险库加解密；
- Wallet Core 进行派生和本地签名；
- HTTP/SSE 调用本地 Go 服务；
- 当前 Trace/Span 机制跟踪请求。

### 19.2 统一平台适配层

业务组件不能直接依赖 IndexedDB、Keychain、Keystore 或 `chrome.storage`，而应通过统一接口访问：

```text
普通浏览器    -> IndexedDB + Web Crypto
iOS          -> Keychain + Capacitor Plugin
Android      -> Android Keystore + Capacitor Plugin
Chrome 插件  -> chrome.storage.local/session + Web Crypto
```

页面、账户、资产、Send、Receive、History 和 Agent 逻辑只依赖统一接口，不关心具体运行平台。

### 19.3 推荐开发命令

改造后应提供统一命令：

```bash
npm run dev:web
npm run build:web
npm run build:mobile
npm run sync:ios
npm run sync:android
npm run build:extension
go run ./cmd/web3-service-agent
```

日常开发流程建议：

1. 启动本地 Go 服务；
2. 在普通浏览器验证绝大多数钱包和 Agent 功能；
3. 只有涉及 Keychain、Keystore、生物识别、相机和 Deep Link 时才启动移动端模拟器或真机；
4. 只有涉及 Manifest V3、Service Worker、`chrome.storage` 时才加载 Chrome 插件；
5. 共享业务逻辑必须优先通过普通浏览器自动测试。

### 19.4 可在普通浏览器验证的功能

- 创建和导入钱包；
- BIP39/BIP44 派生；
- 多账户和多链地址；
- 钱包密码和自动锁定；
- 资产余额；
- History；
- Send 和 Receive；
- 手续费预估；
- 交易详情；
- 本地签名和广播；
- Agent 对话；
- Trace 和 Span；
- 页面布局和绝大多数交互。

### 19.5 必须在平台环境验证的功能

| 功能 | 验证环境 |
| --- | --- |
| Face ID/Touch ID | iOS 模拟器或真机 |
| iOS Keychain | iOS 模拟器或真机 |
| Android Keystore | Android 模拟器或真机 |
| StrongBox | 支持 StrongBox 的 Android 真机 |
| BiometricPrompt | Android 模拟器或真机 |
| 后台快照遮罩 | iOS/Android |
| Deep Link/App Link | iOS/Android |
| Chrome Service Worker 生命周期 | Chrome 插件 |
| `chrome.storage` | Chrome 插件 |
| DApp Provider | Chrome 插件和测试 DApp |
| EIP-1193/EIP-6963 | Chrome 插件和测试 DApp |

### 19.6 本地 API 和 CORS

优先让 Web 页面继续由 Go 服务托管，以保持同源请求。如果未来使用独立前端开发服务器，例如：

```text
Web UI：http://localhost:5173
Go API：http://localhost:8081
```

推荐由前端开发服务器代理 `/v1`、`/wallet` 和 `/ui` 到 Go 服务，而不是在生产环境开放通配符 CORS。

如果确实需要跨域，应只允许明确的开发来源，例如：

```text
http://localhost:5173
http://127.0.0.1:5173
chrome-extension://实际插件ID
```

生产环境不得对包含身份信息或钱包接口的请求使用无限制的 `Access-Control-Allow-Origin: *`。

### 19.7 验收标准

多端改造完成后，应满足：

- 不安装 Xcode、Android Studio 或 Chrome 插件也能运行 Web 回归测试；
- Go 内嵌页面可以直接访问；
- Web 与移动端、插件共用 Wallet Core 测试向量；
- 平台专属代码集中在适配器和原生插件中；
- 浏览器回归测试失败时，移动端和插件构建不得发布；
- Web 本地验证能力不得因为移动端改造而退化。

---

## 20. 当前实施状态

截至 2026-09-19，第一轮多端工程改造已经开始，当前状态如下。

### 已完成

- [x] 保留 Go 内嵌 Web 页面和 `http://localhost:8081` 验证方式；
- [x] 增加统一平台 Runtime；
- [x] 增加可配置 API Base URL；
- [x] Web 继续使用 IndexedDB 保险库；
- [x] Chrome 使用 `chrome.storage.local` 保存加密保险库；
- [x] 创建 Capacitor 8 移动端工程；
- [x] 生成 iOS Xcode 工程；
- [x] 生成 Android Gradle 工程；
- [x] iOS 接入 Keychain 加密保险库存储；
- [x] Android 接入 Android Keystore 和 AES-GCM 二次保护；
- [x] Android 关闭钱包数据自动备份；
- [x] Android 敏感页面启用 `FLAG_SECURE`；
- [x] 创建 Chrome Manifest V3 插件；
- [x] Chrome 提供 Popup 和 Side Panel；
- [x] Chrome 当前不注入 DApp Provider；
- [x] 构建过程自动移除内联脚本，满足 Manifest V3 CSP；
- [x] 增加本地开发 CORS，并限制为 loopback、Android Emulator 或显式 Origin；
- [x] 增加 Web、Mobile、Extension 统一构建命令；
- [x] 增加客户端构建测试和 Go 回归测试。

### 下一阶段

- [x] 将单体 `index.html` 拆出 HTML、CSS、启动脚本和应用逻辑；
- [x] 正式抽取 TypeScript Wallet Core 包；
- [x] 正式抽取 TypeScript API Client 包；
- [x] 建立 TypeScript UI Component 基础包；
- [x] 抽取 TypeScript BIP44 账户与路径管理模块；
- [x] 抽取 TypeScript 历史交易归一化、合并和筛选模块；
- [x] 抽取 TypeScript 交易校验与签名入参模块；
- [x] 抽取 TypeScript 应用初始状态和行情计算模块；
- [x] 抽取 History 页面控制器（缓存、筛选、链上同步、渲染和事件）；
- [x] 抽取跨端二维码矩阵生成与 SVG 渲染模块；
- [x] 抽取 Send/Receive 控制器（Token 选择、最大值、手续费预估、确认、签名广播和收款二维码）；
- [x] 抽取 Asset Balance 控制器（链映射、地址派生缓存、余额同步和资产列表渲染）；
- [x] 抽取 BIP44 Account 控制器（账户列表、切换、添加、重命名和加密 Keyring 持久化）；
- [x] 抽取 Biometric 控制器（能力检测、启用选择、快捷解锁和关闭生物识别）；
- [x] 抽取跨端 Wallet Vault Store（Web IndexedDB、iOS Keychain、Android Keystore 路由）；
- [x] 抽取 Wallet Session 控制器（30 分钟续期、后台锁定和 Seed 内存清零）；
- [x] 抽取 Secret Export 控制器（助记词/私钥二次验证、当前 BIP44 账户派生、复制和 60 秒清除）；
- [x] 抽取 Wallet Vault 控制器（创建、助记词导入、解锁、修改密码和锁屏弹窗状态）；
- [x] iOS 接入 Face ID/Touch ID 快捷解锁；
- [x] Android 接入 BiometricPrompt 快捷解锁；
- [x] iOS 增加后台快照遮罩；
- [ ] 增加移动端 API 地址设置页面；
- [ ] 增加二维码相机扫描；
- [ ] 增加 Deep Link/App Link；
- [x] 使用 Xcode 27 / iOS 27 SDK 完成模拟器目标编译；
- [x] 使用 Android SDK 36 完成单元测试和 Debug APK 编译；
- [x] 将钱包创建、导入、解锁和修改密码流程拆为 Wallet Vault 控制器；
- [ ] 为 Chrome Web Store 准备固定 Extension ID、图标、隐私说明和生产 API Origin；
- [ ] 增加真实设备端到端测试。
