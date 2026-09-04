# hello

[![test](https://github.com/soulteary/hello/actions/workflows/test.yml/badge.svg)](https://github.com/soulteary/hello/actions/workflows/test.yml)
[![docker](https://github.com/soulteary/hello/actions/workflows/docker.yml/badge.svg)](https://github.com/soulteary/hello/actions/workflows/docker.yml)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![Go Version](https://img.shields.io/github/go-mod/go-version/soulteary/hello)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

![Hello!](.github/assets/hello.jpg)

一个围绕经典 Party Parrot 构建的轻量终端动画与 HTTP 演示程序。它既可以
作为 `docker/hello-world` 的有趣替代品，也适合用于容器、反向代理、鉴权网关和
健康检查演示。

[English README](https://github.com/soulteary/hello/blob/main/README.md)

## 快速开始

运行终端动画：

```bash
docker run --rm -it soulteary/hello
```

默认动画会持续循环，按 `Ctrl+C` 退出。如果希望容器播放一次后自动退出：

```bash
docker run --rm soulteary/hello -loops 1
```

镜像同时发布到 Docker Hub 与 GitHub Container Registry：

```bash
docker run --rm -it soulteary/hello
docker run --rm -it ghcr.io/soulteary/hello
```

需要可重复部署时，应固定完整版本，例如
`ghcr.io/soulteary/hello:2.2.0`，或直接固定不可变 digest。`latest` 跟随最近一次
正式发布，`main` 跟随默认分支。完整示例见[部署指南](docs/deployment.md)和
[校验指南](docs/verification.md)。

## HTTP 模式

显式启动内置 HTTP 服务：

```bash
docker run --rm -p 8080:8080 ghcr.io/soulteary/hello -listen :8080
```

根路径会根据请求自动选择响应形式：

- `curl` 和其他非浏览器客户端得到 `text/plain`：从所选动画中随机取一帧原始
  ASCII 图案，后面是有助于排查代理链路且不包含凭据的请求摘要。服务启动时会
  缓存去重后的全部帧，请求期间只需在内存中完成选择；响应最后一行标明项目
  的规范地址。
- 浏览器得到终端风格 HTML 页面。页面连接 `/events` 并保留代理对外路径前缀，
  服务端通过 Server-Sent Events（SSE）持续推送动画帧；JavaScript 在每帧到达后
  刷新 `<pre>` 内容，页面底部同时展示项目地址文本链接。
- `?format=text` 和 `?format=html` 可覆盖自动判断；`plain` 是 `text` 的别名。

分别测试文本和 HTML 响应：

```bash
curl http://127.0.0.1:8080/
curl -I http://127.0.0.1:8080/
curl http://127.0.0.1:8080/healthz

# 在非浏览器客户端中强制查看 HTML
curl 'http://127.0.0.1:8080/?format=html'
```

然后用浏览器打开 <http://127.0.0.1:8080/>，即可看到自动刷新的终端鹦鹉。
如果 JavaScript 被关闭或浏览器不支持 SSE，页面仍会显示内嵌的第一帧。

### HTTP 端点

| 端点 | 方法 | 响应 |
| --- | --- | --- |
| `/` | `GET`、`HEAD` | 自动协商的文本帧或 HTML 终端。 |
| `/events` | `GET`、`HEAD` | 浏览器页面使用的 SSE 动画流。 |
| `/healthz` | `GET`、`HEAD` | `200 OK` 与 `ok`。 |

`/events` 中的事件名是 `frame`，数据为 JSON：

```text
event: frame
data: {"animation":"parrot","color":"#ff8787","frame":"...","index":0}
```

完整事件还包含递增的 `id`、一秒重连提示；服务端每 15 秒发送一次注释心跳：

```text
id: 0
event: frame
retry: 1000
data: {"animation":"parrot","color":"#ff8787","frame":"...","index":0}

: keepalive
```

这里的 SSE 是建立在普通长连接 HTTP 响应之上的应用层 Server Push。项目没有
采用 HTTP/2 Resource Push，因为现代浏览器已经不再将它作为通用页面资源推送
机制支持。

### HTTP 选项

终端参数会在含义合理时复用于 HTTP 模式：

```bash
# 以 120 ms 帧间隔展示 cat，并关闭彩色循环
docker run --rm -p 8080:8080 ghcr.io/soulteary/hello \
  -listen :8080 -animation cat -delay 120 -mono
```

- `-animation` / `-a` 决定终端客户端和浏览器动画使用的帧集合。
- `-delay` 决定 SSE 推送帧间隔。
- `-mono` 关闭浏览器颜色循环。
- `-http-max-streams` 限制并发 SSE 客户端，默认值为 `64`。
- `-loops`、`-list` 不能与 `-listen` 组合使用。

服务设置了请求和请求头超时，并支持 `SIGINT` / `SIGTERM` 优雅退出。SSE 每次
写入使用独立截止时间，因此不会被较短的全局响应超时误切断。若 `/events` 前方
存在反向代理，需要为该路径关闭响应缓冲；hello 同时会发送兼容代理可识别的
`X-Accel-Buffering: no`。

全部流槽位占满时，新的 `/events` 请求会得到 `503 Service Unavailable` 和
`Retry-After: 1`，但 `/healthz` 不受影响。浏览器页面在隐藏时会主动断开 SSE，
提供暂停/继续按钮，并在系统偏好“减少动态效果”时默认保持暂停。

### 请求信息回显范围

文本响应包含方法、URL 路径、Host、容器主机名和版本。问候语与版本之间保留一个
空行，最后一行固定为 `Project: https://github.com/soulteary/hello`。根路径响应
（包括 `HEAD`）还会在 `Project` 响应头中提供同一地址。根据显式诊断参数，正文
只能输出以下允许列表内的路由或身份请求头：

- `Forwarded`、`User-Agent`、`X-Real-IP`
- `X-Forwarded-For`、`X-Forwarded-Host`、`X-Forwarded-Method`、
  `X-Forwarded-Port`、`X-Forwarded-Prefix`、`X-Forwarded-Proto`、
  `X-Forwarded-URI`、`X-Forwarded-User`
- `X-Auth-Email`、`X-Auth-Groups`、`X-Auth-Name`、`X-Auth-Role`、
  `X-Auth-Scopes`、`X-Auth-User`

凭据类或未识别的请求头不会回显，包括 `Authorization`、
`Proxy-Authorization`、`Cookie`、`X-Auth-Token` 和
`X-Forwarded-Access-Token`。默认也不会输出 `X-Forwarded-User`、允许列表内的
`X-Auth-*` 身份字段或 URL 查询参数。仅应在受控测试环境中启用
`-reflect-identity` 或 `-reflect-query`。不希望暴露基础设施标识时，可使用
`-reflect-hostname=false` 关闭默认主机名输出。反向代理必须先清除客户端自行
提交的身份头，再写入可信值。

## 内置动画

| 名称 | 描述 |
| --- | --- |
| `parrot` | 经典 Party Parrot。 |
| `cat` | 蹦跶的小猫。 |
| `coffee` | 一杯冒着热气的咖啡。 |
| `loading` | 简易加载转圈。 |
| `pedro` | 浣熊 Pedro。 |

动画名可作为位置参数，也可通过 `-a` / `-animation` 传入：

```bash
docker run --rm -it soulteary/hello cat
docker run --rm -it soulteary/hello -a coffee
docker run --rm soulteary/hello -list
```

新增动画或了解元数据、分帧规则，请阅读
[动画格式说明](docs/animation-format.md)。

## 命令行参数

| 参数 | 描述 | 默认值 |
| --- | --- | --- |
| `-a`、`-animation` | 动画名；优先于位置参数。 | `""` |
| `-loops` | 终端循环次数；`0` 表示无限。 | `0` |
| `-delay` | 帧间隔（毫秒）；范围 `1`–`60000`。 | `75` |
| `-mono` | 关闭彩虹色。 | `false` |
| `-list` | 列出内置动画并退出。 | `false` |
| `-listen` | 监听 HTTP，而不是进入终端播放循环。 | `""` |
| `-http-max-streams` | HTTP 模式允许的最大并发 SSE 客户端数。 | `64` |
| `-reflect-query` | 在文本诊断中输出原始 URL 查询参数。 | `false` |
| `-reflect-identity` | 在文本诊断中输出允许列表内的身份头。 | `false` |
| `-reflect-hostname` | 在文本诊断中输出运行时主机名。 | `true` |
| `-version` | 打印构建版本并退出。 | `false` |
| `-h`、`-help` | 打印使用说明并退出。 | `false` |

最多接受一个位置动画参数。动画名不存在、参数越界或参数组合冲突时，程序会向
stderr 输出明确原因，并以非零状态码退出。

## 安装二进制

使用 [`go.mod`](go.mod) 声明的 Go 版本从源码安装：

```bash
go install github.com/soulteary/hello/cmd/hello@latest
```

需要可重复安装时，应固定为 `@v2.2.0`，而不是使用 `@latest`。

从 `v2.0.0` 开始，正式版本包含 SHA-256 校验文件，以及以下预编译二进制：

- Linux：amd64、arm64
- macOS：amd64、arm64
- Windows：amd64、arm64

发布压缩包会同时包含 `LICENSE` 和 `NOTICE`。容器镜像使用非 root 用户运行在
distroless 基础镜像中，并在 `/usr/share/licenses/hello/` 保留相同文件。

已经发布的 `v2.1.0` GitHub Release 不完整，没有可下载的二进制压缩包。后续完整
版本发布前，请使用 `v2.0.0` 预编译产物；该事件已记录在
[变更记录](docs/CHANGELOG.md)中。

## 终端兼容性

终端动画依赖 ANSI 光标控制与 256 色转义序列。如果终端不支持，请使用
`-mono`；也可增加 `-loops 1`，避免无法正确显示的动画无限运行。

Windows 建议使用 Windows Terminal 或较新的 PowerShell。旧版 `cmd.exe`
可能无法正确渲染颜色和光标控制序列。

## 开发

项目是一个无第三方运行时依赖的 Go 模块，按职责拆分为以下内部包：

| 路径 | 职责 |
| --- | --- |
| `cmd/hello` | 参数解析，以及终端 / HTTP 模式分发。 |
| `internal/animation` | 内嵌动画清单与格式解析。 |
| `internal/render` | ANSI 终端渲染。 |
| `internal/cli` | 终端播放循环与信号处理。 |
| `internal/httpserver` | 内容协商、HTML、SSE 与健康检查。 |

常用命令：

```bash
make help          # 查看所有目标
make build         # 构建带版本信息的 ./hello
make test          # 使用 race detector 运行全部测试
make cover         # 运行测试并执行 100% 语句覆盖率门槛
make lint          # 运行必需的 golangci-lint
make vuln          # 使用 govulncheck 扫描可达代码
make check         # 运行全部必需的本地质量门槛（含模块整洁性）
make fuzz          # 对动画解析器模糊测试 30 秒
make bench         # 运行渲染器基准测试
make docker        # 构建本地容器镜像
```

默认门槛和 CI 门槛均为 100%。新增成功或失败路径时应补齐测试，而不是降低
`COVERAGE_MIN`。

CI 会检查模块整洁性、格式、`go vet`、govulncheck、golangci-lint、竞态测试和
覆盖率门槛，还会实际启动构建出的容器，检查非 root 用户、OCI 元数据、HTTP
行为和默认诊断脱敏。标签发布会在上传前校验压缩包内容、checksum 和可执行文件
版本。CodeQL 在 push、PR 和每周定时任务中执行。第三方 GitHub Actions 均固定
到完整 Commit SHA。

贡献流程见 [CONTRIBUTING.md](.github/CONTRIBUTING.md)，私密安全问题报告方式见
[SECURITY.md](.github/SECURITY.md)，HTTP 兼容性变化见
[v2 迁移指南](docs/migration-v2.md)，代理和 Kubernetes 示例见
[部署指南](docs/deployment.md)。维护者在创建任何版本标签前，应执行
[发布指南](docs/releasing.md)。

## 致谢与许可证

项目基于 [John Hobbs](https://github.com/jmhobbs) 在 2016 年发布的
[jmhobbs/terminal-parrot](https://github.com/jmhobbs/terminal-parrot) 深度重构。

项目使用 [MIT 许可证](LICENSE)：

- Copyright (c) 2016 John Hobbs —— 原始作品
- Copyright (c) 2026 soulteary —— 后续修改与新增内容

再次分发源码、发布压缩包或容器镜像时，必须保留 `LICENSE` 和 `NOTICE`。
内置 ASCII 素材的完整署名见 [`NOTICE`](NOTICE)。
