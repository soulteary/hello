# Migrating to hello 2.0

[English](#english) | [中文](#中文)

## English

Version 2.0 keeps terminal playback compatible, but deliberately changes the
HTTP root response so the same container is useful from both a shell and a
browser.

### HTTP behavior changes

| Request | Before 2.0 | 2.0 |
| --- | --- | --- |
| `curl /` | Plain request summary. | First ASCII animation frame, blank line, then the request summary. |
| Browser `GET /` | Plain request summary. | HTML terminal that consumes the `/events` SSE stream. |
| `HEAD /` | `200`, text content type. | `200`, with the content type selected by the same negotiation rules as `GET`. |
| `GET /events` | Not available. | Long-lived `text/event-stream` response. |
| `GET /healthz` | `ok`. | Unchanged. |

Clients that parse the root body by line number must be updated because the
ASCII frame now precedes `Hello from soulteary/hello!`. Prefer matching the
named diagnostic fields, or use `/healthz` when only availability matters.

Representation selection follows these rules, in order:

1. `?format=html` or `?format=text` explicitly selects the response.
2. A `curl/*` user agent receives text unless the query override is present.
3. A positive `text/html` / `application/xhtml+xml` `Accept` value receives
   HTML.
4. A browser-like `Mozilla/*` user agent receives HTML.
5. Every other client receives text.

An unsupported `format` value now returns `400 Bad Request` instead of being
ignored.

### Header reflection is narrower

Earlier HTTP mode reflected every `X-Forwarded-*` and `X-Auth-*` header. That
could expose a deployment-specific credential such as
`X-Forwarded-Access-Token` or `X-Auth-Token`. Version 2.0 uses an explicit
identity/routing allowlist documented in the README. If a demo relied on a
custom reflected header, rename it to a documented non-secret field or inspect
it at the proxy instead. Do not add token or authorization fields to the
allowlist.

### HTTP flags

`-animation`, `-delay` and `-mono` now configure HTTP output as well as terminal
playback. `-loops` is terminal-only and is rejected when `-listen` is present;
`-list` remains incompatible with `-listen`.

Only one positional animation name is accepted in all modes. Version 1.x
silently ignored additional positional arguments; version 2.0 exits with code
2 so mistakes are visible.

### Reverse-proxy configuration

`/events` is a streaming response. A proxy in front of hello must:

- forward the response without buffering;
- keep the upstream read timeout longer than the desired session;
- avoid response transformations that aggregate chunks;
- propagate client disconnects so the server can release the stream promptly.

hello sends `Cache-Control: no-store, no-transform` and
`X-Accel-Buffering: no`, but proxy configuration remains authoritative.
`/healthz` should be used for health checks rather than `/events`.

### Packaging and CI

Release archives now contain `LICENSE` and `NOTICE`, Windows arm64 is included,
and the runtime image stores notices in `/usr/share/licenses/hello/`. A release
tag is tested, linted and checked against the coverage floor before artifacts
are built.

---

## 中文

2.0 保持终端播放方式兼容，但有意调整了 HTTP 根路径，使同一个容器既适合命令行
访问，也能在浏览器中直接展示动画。

### HTTP 行为变化

| 请求 | 2.0 之前 | 2.0 |
| --- | --- | --- |
| `curl /` | 纯文本请求摘要。 | 第一帧 ASCII 动画、一个空行，然后是请求摘要。 |
| 浏览器 `GET /` | 纯文本请求摘要。 | 消费 `/events` SSE 流的 HTML 终端。 |
| `HEAD /` | `200`，文本类型。 | `200`，Content-Type 与 `GET` 使用相同协商规则。 |
| `GET /events` | 不存在。 | 长连接 `text/event-stream` 响应。 |
| `GET /healthz` | `ok`。 | 保持不变。 |

如果客户端按固定行号解析根路径，需要调整逻辑，因为 ASCII 帧会出现在
`Hello from soulteary/hello!` 之前。建议根据字段名称匹配；如果只检查可用性，
应改用 `/healthz`。

响应形式按以下优先级判断：

1. `?format=html` 或 `?format=text` 显式指定响应。
2. `curl/*` User-Agent 默认获得文本，除非存在查询参数覆盖。
3. `Accept` 中存在权重大于 0 的 `text/html` 或
   `application/xhtml+xml` 时返回 HTML。
4. 类似 `Mozilla/*` 的浏览器 User-Agent 返回 HTML。
5. 其他客户端返回文本。

不支持的 `format` 值现在返回 `400 Bad Request`，不再静默忽略。

### 请求头回显范围收紧

早期 HTTP 模式会回显全部 `X-Forwarded-*` 和 `X-Auth-*` 请求头，这可能暴露
部署中自定义的 `X-Forwarded-Access-Token` 或 `X-Auth-Token`。2.0 改为 README
中列出的明确身份与路由请求头白名单。如果演示依赖自定义回显字段，请改用已有
的非敏感字段，或直接在代理层检查；不要把令牌、授权字段加入白名单。

### HTTP 参数

`-animation`、`-delay`、`-mono` 现在同时配置 HTTP 输出与终端播放。
`-loops` 仅适用于终端，与 `-listen` 同时出现时会被拒绝；`-list` 仍不能与
`-listen` 组合。

所有模式最多接受一个位置动画参数。1.x 会静默忽略额外位置参数；2.0 会返回
状态码 2，让输入错误显式可见。

### 反向代理配置

`/events` 是流式响应。hello 前方的代理需要：

- 关闭响应缓冲；
- 将上游读取超时设置为大于预期会话时长；
- 避免聚合响应块的内容转换；
- 传递客户端断开事件，让服务端及时释放流。

hello 会发送 `Cache-Control: no-store, no-transform` 与
`X-Accel-Buffering: no`，但最终仍以代理配置为准。健康检查应使用 `/healthz`，
不要连接 `/events`。

### 打包与 CI

发布压缩包现在包含 `LICENSE`、`NOTICE`，增加 Windows arm64，并在运行时镜像
的 `/usr/share/licenses/hello/` 保存许可证文件。标签触发构建前会先运行测试、
lint 和覆盖率门槛检查。
