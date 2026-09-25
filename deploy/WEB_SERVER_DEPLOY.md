# Web + Go 服务同机部署

压缩包里的 `web3-service-agent` 已内嵌 Web 页面，不需要另外部署 `dist/web`。仅适用于与压缩包名称匹配的 Linux CPU 架构。

1. 将压缩包上传服务器并解压到专用目录。进入解压出的 `web3agent` 目录。
2. 复制 `etc/application.example.properties` 为 `etc/application.properties`，按服务器实际情况调整模型、行情源和 RPC。不要在配置文件或 Web 目录中写入明文密钥。
   `service.aiEnabled=false` 默认关闭 Home 页 AI 聊天；改为 `true` 并重启服务后才开放聊天接口与输入框。
3. 通过服务管理器提供 `DEEPSEEK_API_KEY`、`COINGECKO_API_KEY` 和 `AGENT_WALLET_ALLOWED_ORIGINS=https://你的域名`。若使用其他模型，配置相应的环境变量。服务的工作目录必须是本目录，否则找不到 `etc/application.properties`。
4. 执行 `./start.sh` 启动服务。脚本会先检查同名进程，已经启动时会报错并拒绝重复启动。使用 `./stop.sh` 停止服务，使用 `./restart.sh` 重启服务。然后确认本机 `http://127.0.0.1:8081/healthz` 返回 `{"status":"ok"}`。
5. 使用 Nginx/Caddy 等把同一 HTTPS 域名的 `/`、`/ui/`、`/wallet/`、`/v1/` 请求都反向代理到 `http://127.0.0.1:8081`。代理必须传递原始 Host 和协议，例如 Nginx 设置 `proxy_set_header Host $host;`、`proxy_set_header X-Forwarded-Host $host;`、`proxy_set_header X-Forwarded-Proto $scheme;`。不要单独把 `dist/web` 放到另一个域名。不要对公网开放 8081 端口。

当前 Go 服务监听 `:8081`（所有网卡）；请用防火墙或容器网络限制 8081 只供反向代理访问。HTTPS 终止于代理时，后端必须设置上述精确的 `AGENT_WALLET_ALLOWED_ORIGINS`，否则浏览器 POST 请求可能被判为不允许的来源。

上线后打开 `https://你的域名/`，并实际检查资产、History、发送确认等功能。钱包保存在浏览器本机；更换域名或浏览器不会自动迁移本地保险库，请妥善备份助记词。
