# hello

[![test](https://github.com/soulteary/hello/actions/workflows/test.yml/badge.svg)](https://github.com/soulteary/hello/actions/workflows/test.yml)
[![docker](https://github.com/soulteary/hello/actions/workflows/docker.yml/badge.svg)](https://github.com/soulteary/hello/actions/workflows/docker.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/hello)](https://goreportcard.com/report/github.com/soulteary/hello)
[![Go Version](https://img.shields.io/github/go-mod/go-version/soulteary/hello)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

`docker/hello-world` 的彩虹鹦鹉替代品。

[English README](README.md)

## 使用方式

```bash
docker run --rm soulteary/hello
```

或从 GitHub Container Registry 拉取：

```bash
docker run --rm ghcr.io/soulteary/hello
```

或使用 Go 直接安装二进制：

```bash
go install github.com/soulteary/hello@latest
```

Linux、macOS、Windows（amd64/arm64）的预编译二进制可在每个
[GitHub Release](https://github.com/soulteary/hello/releases) 中下载。

示例：

```bash
# 默认：经典 Party Parrot，无限循环
docker run --rm soulteary/hello

# 跑 3 圈后退出
docker run --rm soulteary/hello -loops 3

# 切换动画并关闭彩虹色
docker run --rm soulteary/hello -mono cat
```

## 内置动画

| 名称      | 描述                |
| --------- | ------------------- |
| `parrot`  | 经典 Party Parrot。 |
| `cat`     | 蹦跶的小猫。        |
| `coffee`  | 一杯冒着热气的咖啡。|
| `loading` | 简易加载转圈。      |
| `pedro`   | 浣熊 Pedro。        |

动画名作为位置参数传入，例如
`docker run --rm soulteary/hello cat`。不传则默认 `parrot`。

若想了解或新增动画，`*.animation` 文件格式见
[`docs/animation-format.md`](docs/animation-format.md)。

列出内置动画（每个名字后会显示 `description` 元数据）：

```bash
$ docker run --rm soulteary/hello -list
cat     A bouncing cat
coffee  A steaming cup of coffee
loading A simple loading spinner
parrot  The classic Party Parrot.
pedro   Pedro the raccoon
```

## 参数

| 参数         | 描述                                  | 默认值  |
| ------------ | ------------------------------------- | ------- |
| `-a`, `-animation` | 动画名（覆盖位置参数）。        | `""`    |
| `-loops`     | 循环次数，`0` 表示无限。              | `0`     |
| `-delay`     | 帧间隔（毫秒，必须 > 0）。            | `75`    |
| `-mono`      | 关闭彩虹色，输出单色。                | `false` |
| `-list`      | 列出所有内置动画并退出。              | `false` |
| `-version`   | 打印版本并退出。                      | `false` |
| `-h`, `-help` | 打印用法并退出。                     | `false` |

## 注意事项

输出依赖 ANSI 转义序列。如果你的终端不支持，画面会错乱 —— 这种情况下建议加上
`-loops 1`，让它跑完一轮就退出，而不是无限循环。

在 Windows 上请使用 [Windows Terminal](https://aka.ms/terminal) 或较新的
PowerShell；旧版 `cmd.exe` 控制台可能无法正确渲染 256 色序列。

## 开发

本项目是一个单文件 Go 模块，无第三方依赖。

```bash
make help         # 列出所有可用目标
make build        # 构建 ./hello 二进制
make test         # 带 -race 的测试
make cover        # 测试并打印覆盖率
make check        # gofmt + vet + test（与 CI 一致）
make docker       # 本地构建 Docker 镜像
```

CI 会在每次 push / PR 时运行 `go vet`、`gofmt -l`、`go test -race`
（`.github/workflows/test.yml`）。Docker 镜像会从 `main` 分支与 `v*` tag
触发构建并发布（`.github/workflows/docker.yml`）。

## 致谢

本项目基于
[jmhobbs/hello-parrot](https://github.com/jmhobbs/hello-parrot)（作者
[John Hobbs](https://github.com/jmhobbs)，2016 年）深度重构而来。

感谢原作者带来的彩虹鹦鹉。当前版本在此基础上新增了 Docker 打包、更多动画、
可扩展的动画加载机制、命令行参数与完整的测试套件。

## 参与贡献与安全

欢迎贡献 —— 开发流程与新增动画的检查清单见
[`CONTRIBUTING.md`](.github/CONTRIBUTING.md)。报告安全问题请参考
[`SECURITY.md`](.github/SECURITY.md)。所有参与者都应遵守
[行为准则](.github/CODE_OF_CONDUCT.md)。

## 许可证

基于 [MIT 许可证](LICENSE) 发布。

- Copyright (c) 2016 John Hobbs —— 原始作品
- Copyright (c) 2026 soulteary —— 后续修改与新增内容

再次分发本项目时（包括二进制与 Docker 镜像），请同时保留 `LICENSE` 和
`NOTICE` 文件，以确保所有版权与署名信息都得到完整传递 —— 这是 MIT 许可证
的硬性要求。完整的署名清单（含 `internal/animation/assets/animations/` 下第三方 ASCII 素材）见
[`NOTICE`](NOTICE)。
