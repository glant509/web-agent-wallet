# Agent Wallet 客户端工程

该目录包含移动端和 Chrome 插件外壳。客户端静态资源统一从 `internal/httpapi/ui` 生成，现有 Go 内嵌 Web 页面仍然是可运行、可回归的基线。

## 生成全部客户端

在仓库根目录执行：

```bash
npm install
npm run build:clients
```

生成结果位于：

```text
dist/web
dist/mobile
dist/extension
```

`dist` 是生成目录，不提交到 Git。

共享 TypeScript 源码位于 `clients/shared/src`，目前包含：

- `wallet-core`：保险库、KDF、BIP39 seed、编码和安全随机数；
- `wallet-accounts`：网络、BIP44 Keyring、账户路径和账户名称；
- `wallet-history`：历史交易归一化、去重、合并与筛选；
- `controllers/wallet-history`：History 页面的缓存、筛选、同步、渲染和事件控制；
- `controllers/asset-transfer`：Send/Receive 的 Token 选择、校验、手续费预估、确认、签名广播和收款交互；
- `controllers/asset-balance`：资产链映射、地址缓存、余额同步和资产列表渲染；
- `controllers/wallet-accounts`：BIP44 账户列表、切换、添加、重命名和 Keyring 保险库持久化；
- `controllers/wallet-biometrics`：生物识别能力检测、启用选择、快捷解锁和关闭控制；
- `wallet-vault-store`：根据运行平台路由 IndexedDB、iOS Keychain 或 Android Keystore；
- `controllers/wallet-session`：自动锁定、活动续期、后台锁定和敏感内存清理；
- `controllers/wallet-secret-export`：助记词/当前账户私钥的二次验证、BIP44 路径展示、复制和 60 秒自动清除；
- `controllers/wallet-vault`：钱包创建、助记词导入、解锁、修改密码以及锁屏弹窗状态编排；
- `transactions`：地址和金额校验、签名交易参数整理；
- `market`：K 线解析、涨跌幅和图表数据格式化；
- `qr-code`：Receive 页面二维码矩阵生成和 SVG 渲染；
- `app-state`：跨端一致的应用初始状态；
- `api-client`、`ui-components`、`biometric-auth`：接口访问、通用 UI 逻辑和生物识别桥接。

`npm run typecheck` 执行严格类型检查，`npm run build:shared` 生成供 Web、iOS、Android 和 Chrome 插件共用的 `shared.js`。

## 本地 Web

```bash
npm run dev:web
```

默认访问：

```text
http://localhost:8081
```

也可以继续直接使用：

```bash
go run ./cmd/web3-service-agent
```

## iOS

要求：

- 完整 Xcode；
- Xcode Command Line Tools 指向完整 Xcode；
- 可用的 iOS Simulator Runtime。

同步并打开工程：

```bash
npm run sync:ios
npm run open:ios
```

工程位置：

```text
clients/mobile/ios/App/App.xcodeproj
```

iOS 使用 Keychain 保存二次设备保护后的加密钱包保险库，并通过 LocalAuthentication 接入 Face ID/Touch ID 快捷解锁。快捷凭据使用 `biometryCurrentSet` 保护，设备生物信息变更后自动失效；钱包自身的密码加密层仍然保留。

## Android

要求：

- Java 21；
- Android SDK 36；
- Android Studio 或可用的 Android 命令行工具。

同步并打开工程：

```bash
npm run sync:android
npm run open:android
```

工程位置：

```text
clients/mobile/android
```

Android 使用 Android Keystore 生成不可直接导出的 AES 密钥，以 AES-GCM 二次保护加密钱包保险库，并关闭系统自动备份。
快捷解锁使用 AndroidX `BiometricPrompt` 和 `BIOMETRIC_STRONG`，凭据密钥会在新增生物信息后失效。用户必须先使用钱包密码成功解锁并主动勾选，才会保存快捷凭据。

Android 模拟器默认访问：

```text
http://10.0.2.2:8081
```

iOS 模拟器和 Chrome 插件默认访问：

```text
http://localhost:8081
```

真机或生产环境需要设置实际 HTTPS API 地址。平台 Runtime 支持通过以下方式覆盖：

```js
window.AgentWalletPlatform.setAPIBaseURL("https://api.example.com")
```

生产服务端同时要配置精确允许的 Origin：

```bash
export AGENT_WALLET_ALLOWED_ORIGINS="capacitor://localhost,https://localhost,chrome-extension://实际插件ID"
```

## Chrome 插件

生成插件：

```bash
npm run build:extension
```

在 Chrome 中：

1. 打开 `chrome://extensions`；
2. 开启“开发者模式”；
3. 选择“加载已解压的扩展程序”；
4. 选择 `dist/extension`。

当前插件只提供钱包自身的资产、Send、Receive、History 和 Agent 页面，没有 Content Script，也不会向网页注入 `window.ethereum`。

## 安全边界

- 助记词、Seed 和私钥只在客户端短暂存在；
- 服务端只接收地址、公开交易参数和已签名交易；
- 移动端退到后台时 Web 页面会立即锁定并清理 Seed；
- Android 使用 `FLAG_SECURE` 避免敏感页面出现在截图和任务预览中；
- Chrome 插件使用 Manifest V3，不包含远程可执行代码；
- Chrome 加密保险库只保存在 `chrome.storage.local`，不进入同步存储；
- 当前阶段不提供 DApp 接入和跨链功能。
