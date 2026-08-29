# Animation file format

[English](#english) | [中文](#中文)

## English

Animations live under [`internal/animation/assets/animations/`](../internal/animation/assets/animations) as `*.animation` files and
are embedded into the binary at build time via `go:embed`. The base name of the
file (without the `.animation` suffix) becomes the animation name used on the
command line, e.g. `cat.animation` is played with `hello cat`.

### Structure

A file is made up of a single **metadata header** followed by **two or more
frames**, all separated by a line containing exactly `!--FRAME--!`:

```
<metadata header>
!--FRAME--!
<frame 1>
!--FRAME--!
<frame 2>
...
```

- The first segment (before the first `!--FRAME--!`) is the metadata header.
- Every subsequent segment is one frame of ASCII art.
- At least two frames are required; a single-frame file is rejected because a
  static picture does not need the animation machinery.
- Empty frames are rejected.
- Unix (`\n`), Windows (`\r\n`) and mixed LF/CRLF line endings are normalized
  before parsing. Bare carriage returns (`\r`) are rejected.

### Metadata header

The metadata header is a set of `key: value` lines. Lines that do not contain a
colon are ignored. Recommended keys:

| Key           | Description                                          |
| ------------- | ---------------------------------------------------- |
| `description` | Short human-readable summary, shown by `hello -list`. |
| `author`      | Who created the artwork.                              |
| `source`      | Upstream URL, if the artwork is derived from one.    |
| `license`     | License of the artwork (e.g. `MIT`).                 |

Only `description` is currently surfaced in the CLI (`-list`); the rest are
documentation/attribution aids. When adding third-party artwork, also update
[`NOTICE`](../NOTICE).

The same inventory is used in HTTP mode. Selecting an animation with
`hello -listen :8080 -animation coffee` changes both the plain-text first frame
and the browser SSE stream.

### Minimal example

A complete two-frame example (see [`coffee.animation`](../internal/animation/assets/animations/coffee.animation)
for a real one):

```
description: a tiny blinking dot
author: you
license: MIT
!--FRAME--!
.
!--FRAME--!
:
```

---

## 中文

动画文件位于 [`internal/animation/assets/animations/`](../internal/animation/assets/animations) 目录下，以 `*.animation` 结尾，并在
构建时通过 `go:embed` 嵌入到二进制中。文件去掉 `.animation` 后缀后的名字即为命令
行使用的动画名，例如 `cat.animation` 通过 `hello cat` 播放。

### 文件结构

文件由一个**元数据头**和**两个及以上的帧**组成，彼此用单独一行的 `!--FRAME--!`
分隔：

```
<元数据头>
!--FRAME--!
<第 1 帧>
!--FRAME--!
<第 2 帧>
...
```

- 第一段（首个 `!--FRAME--!` 之前）是元数据头。
- 之后每一段是一帧 ASCII 画面。
- 至少需要两帧；只有一帧的文件会被拒绝，因为静态图片用不到动画机制。
- 空帧会被拒绝。
- Unix（`\n`）、Windows（`\r\n`）以及混合 LF/CRLF 换行会在解析前统一处理；
  单独出现的回车符（`\r`）会被拒绝。

### 元数据头

元数据头是若干 `key: value` 行，不含冒号的行会被忽略。推荐字段：

| 字段          | 说明                                       |
| ------------- | ------------------------------------------ |
| `description` | 简短描述，`hello -list` 会显示。           |
| `author`      | 素材作者。                                 |
| `source`      | 若衍生自上游，填上游 URL。                 |
| `license`     | 素材许可证（如 `MIT`）。                   |

目前只有 `description` 会在 CLI（`-list`）中展示，其余字段用于文档与署名。引入
第三方素材时，请同时更新 [`NOTICE`](../NOTICE)。

HTTP 模式使用同一份动画清单。例如
`hello -listen :8080 -animation coffee` 会同时改变纯文本首帧和浏览器 SSE 动画流。

### 最小示例

一个完整的两帧示例（真实示例见
[`coffee.animation`](../internal/animation/assets/animations/coffee.animation)）：

```
description: a tiny blinking dot
author: you
license: MIT
!--FRAME--!
.
!--FRAME--!
:
```
