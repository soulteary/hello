# hello

[![test](https://github.com/soulteary/hello/actions/workflows/test.yml/badge.svg)](https://github.com/soulteary/hello/actions/workflows/test.yml)
[![docker](https://github.com/soulteary/hello/actions/workflows/docker.yml/badge.svg)](https://github.com/soulteary/hello/actions/workflows/docker.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/hello)](https://goreportcard.com/report/github.com/soulteary/hello)
[![Go Version](https://img.shields.io/github/go-mod/go-version/soulteary/hello)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A drop-in replacement for `docker/hello-world`, but with a party parrot.

[中文文档 / Chinese README](README.zh-CN.md)

## Usage

```bash
docker run --rm soulteary/hello
```

Or pull from GitHub Container Registry:

```bash
docker run --rm ghcr.io/soulteary/hello
```

Or install the binary directly with Go:

```bash
go install github.com/soulteary/hello@latest
```

Pre-built binaries for Linux, macOS and Windows (amd64/arm64) are attached to
each [GitHub Release](https://github.com/soulteary/hello/releases).

Examples:

```bash
# Default: the classic Party Parrot, looping forever
docker run --rm soulteary/hello

# Run the parrot for exactly 3 loops, then exit
docker run --rm soulteary/hello -loops 3

# Pick a different animation and disable rainbow colors
docker run --rm soulteary/hello -mono cat
```

## Animations

| Name      | Description                       |
| --------- | --------------------------------- |
| `parrot`  | The classic Party Parrot.         |
| `cat`     | A bouncing cat.                   |
| `coffee`  | A steaming cup of coffee.         |
| `loading` | A simple loading spinner.         |
| `pedro`   | Pedro the raccoon.                |

The animation name is passed as a positional argument, e.g.
`docker run --rm soulteary/hello cat`. If omitted, `parrot` is used.

See [`docs/animation-format.md`](docs/animation-format.md) for the
`*.animation` file format if you want to understand or add animations.

List the bundled animations (the `description` metadata is shown alongside each
name):

```bash
$ docker run --rm soulteary/hello -list
cat     A bouncing cat
coffee  A steaming cup of coffee
loading A simple loading spinner
parrot  The classic Party Parrot.
pedro   Pedro the raccoon
```

## Flags

| Flag         | Description                              | Default |
| ------------ | ---------------------------------------- | ------- |
| `-a`, `-animation` | Animation name (overrides positional). | `""`    |
| `-loops`     | Number of loops, `0` for infinite.       | `0`     |
| `-delay`     | Frame delay in milliseconds (must be > 0). | `75`  |
| `-mono`      | Disable rainbow colors.                  | `false` |
| `-list`      | List all available animations and exit.  | `false` |
| `-version`   | Print version and exit.                  | `false` |
| `-h`, `-help` | Print usage and exit.                   | `false` |

## Notes

The output relies on ANSI escape sequences. If your terminal does not support
them, the animation will look garbled — consider running with `-loops 1` so it
exits quickly instead of looping forever.

On Windows, use [Windows Terminal](https://aka.ms/terminal) or a recent
PowerShell; the legacy `cmd.exe` console may not render the 256-color sequences
correctly.

## Development

This project is a single-file Go module with no third-party dependencies.

```bash
make help         # list all available targets
make build        # build the ./hello binary
make test         # run tests with -race
make cover        # run tests and print coverage summary
make check        # gofmt + vet + tests (CI-equivalent)
make docker       # build a local Docker image
```

CI runs `go vet`, `gofmt -l`, `go test -race` on every push and PR
(`.github/workflows/test.yml`). The Docker image is built and published from
`main` and from `v*` tags (`.github/workflows/docker.yml`).

## Credits

This project is a heavily refactored fork of
[jmhobbs/hello-parrot](https://github.com/jmhobbs/hello-parrot) by
[John Hobbs](https://github.com/jmhobbs), originally released in 2016.

Thanks to the original author for the lovely party parrot. The current
distribution adds Docker packaging, additional animations, a pluggable
animation loader, configuration flags, and a full test suite.

## Contributing & Security

Contributions are welcome — see [`CONTRIBUTING.md`](.github/CONTRIBUTING.md) for
the development workflow and the checklist for adding new animations. To report
a security issue, please follow [`SECURITY.md`](.github/SECURITY.md). All
participants are expected to follow the
[Code of Conduct](.github/CODE_OF_CONDUCT.md).

## License

Released under the [MIT License](LICENSE).

- Copyright (c) 2016 John Hobbs — original work
- Copyright (c) 2026 soulteary — modifications and additions

When redistributing this project (including binaries and Docker images), the
`LICENSE` and `NOTICE` files must be included so that all copyright notices
and attribution are preserved, as required by the MIT License. See
[`NOTICE`](NOTICE) for the full attribution list, including third-party
ASCII assets shipped under `animations/`.
