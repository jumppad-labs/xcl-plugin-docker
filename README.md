# xcl Docker plugin

An [xcl](https://github.com/jumppad-labs/xcl) plugin that creates real Docker
networks and containers. It is the example remote plugin: it is built from the
[xcl plugin template](https://github.com/jumppad-labs/xcl-plugin-template-go),
laid out in the standard xcl plugin layout, and published as signed GitHub
releases that xcl's GitHub registry installs.

It provides two block types:

- `docker "network" "<name>"`: a Docker bridge network named after the block,
  with an optional `subnet`. The plugin computes `docker_id`.
- `docker "container" "<name>"`: a Docker container named after the block,
  with `image`, optional `command`, `environment` and `init_script`, and one
  nested `network` block per network it joins. The plugin computes
  `docker_id` and `ip_address`.

```hcl
docker "network" "local" {
  subnet = "10.42.0.0/24"
}

docker "container" "web" {
  image = "nginx:1.27-alpine"

  network {
    name    = docker.network.local.meta.name
    aliases = ["web"]
  }
}
```

`examples/basic/main.xcl` is a complete sample.

## Install

An application installs the plugin from this repository's releases with xcl's
GitHub registry. It downloads the release's index and, with a trusted key,
checks its signature, then downloads only the zip for the current platform,
checks it against its checksum in the index and its manifest against the
index, caches it and runs it as a separate program:

```go
import (
	"github.com/jumppad-labs/xcl"
	"github.com/jumppad-labs/xcl/registry"
)

gh := registry.NewGitHub(registry.GitHubTrustedKeys(publicKey))
gh.RegisterPlugin("jumppad-labs/xcl-plugin-docker", "v0.4.0")

c, err := xcl.NewConfig(xcl.WithRegistry(gh))
```

`publicKey` is the armoured jumppad-labs release key, below. See
[Installing plugins from GitHub](https://github.com/jumppad-labs/xcl#installing-plugins-from-github)
for pinning, private repositories and the cache.

The plugin needs a Docker engine, reached through `DOCKER_HOST` or the default
socket.

## Verifying releases

Releases are signed with the shared jumppad-labs release key, published at
[https://xcl.dev/keys/jumppad-labs-releases.asc](https://xcl.dev/keys/jumppad-labs-releases.asc),
with the fingerprint:

```
15BD A684 3A1F A0EA D1AA  5190 F775 DA00 AFD4 B502
```

Fetch it and check the fingerprint before you trust it:

```shell
curl -fsSLO https://xcl.dev/keys/jumppad-labs-releases.asc
gpg --show-keys jumppad-labs-releases.asc
```

Then pass it to the registry:

```go
publicKey, err := os.ReadFile("jumppad-labs-releases.asc")
if err != nil {
	return err
}

gh := registry.NewGitHub(registry.GitHubTrustedKeys(string(publicKey)))
gh.RegisterPlugin("jumppad-labs/xcl-plugin-docker", "v0.4.0")
```

## Layout

```
plugin.go                       the Plugin type: Init builds the clients once and registers both types
cmd/docker/main.go              serves the Plugin as a separate program
entities/
  network.go                    the docker "network" block type
  container.go                  the docker "container" block type, with its nested network block
providers/
  network.go                    the network provider
  container.go                  the container provider
  attachments.go                previous network attachments and network addresses, used by the container provider
  labels.go                     the labels every Docker object the plugin creates carries
  *_test.go                     strict-double unit tests; docker_test.go runs against a real engine
client/
  docker/docker.go              the narrow interface over the Docker Engine SDK, with New and Ping
  docker/mocks/                 its Mockery-generated double
  containers/containers.go      the container task layer the providers use, built on client/docker
  containers/mocks/             its Mockery-generated double
examples/basic/main.xcl         a sample configuration: a network and a container on it
e2e/                            end-to-end tests: apply the sample, plan with no changes, destroy
.github/workflows/              CI, the weekly check against the latest xcl, and signed releases
.mockery.yml
Makefile
```

- `entities/` holds the block types only. They import nothing but xcl's
  `types`, so a program that only reads what it applied imports `entities`
  alone.
- `providers/` holds the providers. Each is built by a constructor that takes
  the container task layer, so the unit tests build it the same way with a
  double. No provider imports a Docker library.
- `client/` is the only place the Docker libraries are imported.
  `client/docker` mirrors the SDK calls the plugin makes, and
  `client/containers` turns what the providers want done into those calls,
  in its own types.

## Commands

Run each from the repository root:

| Command | What it does |
|---------|--------------|
| `make build` | builds every package, and the plugin binary `build/xcl-plugin-docker` |
| `make test` | runs the unit, real-engine and end-to-end tests |
| `make e2e` | runs only the end-to-end tests |
| `make lint` | runs `go vet`, a gofmt check and staticcheck, as CI does |
| `make generate` | regenerates the Mockery doubles in `client/*/mocks` |
| `make dist VERSION=v0.0.0-snapshot` | builds the release zips and index into `dist/`, unsigned |
| `make release VERSION=v0.4.0` | tags the current commit and pushes the tag |
| `make clean` | removes `build/` and `dist/` |

The real-engine tests in `providers/docker_test.go` and the end-to-end tests
in `e2e/` need a Docker engine. They skip when none is reachable, so
`make test` passes without one.

## Other ways to register the plugin

The `Plugin` type lives in the module's root package, so a host can also
compile it in:

```go
import (
	"github.com/jumppad-labs/xcl"
	docker "github.com/jumppad-labs/xcl-plugin-docker"
	"github.com/jumppad-labs/xcl/registry"
)

local := registry.NewLocal()
local.RegisterPlugin(&docker.Plugin{})

c, err := xcl.NewConfig(xcl.WithRegistry(local))
```

or run a binary built with `make build` as a separate program:

```go
local := registry.NewLocal()
local.RegisterExternalPlugin("build/xcl-plugin-docker")
```

## Reading state

A program that only reads the blocks it applied imports `entities`, and none of
the providers or Docker libraries:

```go
import "github.com/jumppad-labs/xcl-plugin-docker/entities"

containers, err := xcl.FindByType[entities.Container](c, "docker", "container")
```

## Release

```shell
make release VERSION=v0.4.0
```

Tags the current commit and pushes the tag. The release workflow
(`.github/workflows/release.yml`) runs the tests, builds the release with
`make dist`, signs the index with the key in the `GPG_PRIVATE_KEY` and
`GPG_PASSPHRASE` secrets and publishes the release with `gh`. The release
holds:

- `xcl-plugin-docker-<version>-<os>-<arch>.zip` for linux, darwin and windows
  on amd64 and arm64, each with the binary at its root (`.exe` on windows),
  beside `README.md`, `LICENSE` and `xcl-plugin.xcl`, the build's manifest
  naming the plugin, its version, the protocol it speaks, the command to run
  and its platform;
- `xcl-plugin-docker-<version>-index.xcl`, the index listing every zip with
  its platform and its `sha256:` checksum;
- `xcl-plugin-docker-<version>-index.xcl.sig`, an armoured detached signature
  over the index.

Installing it needs xcl v0.13.1 or later, the first release whose GitHub
registry finds a `-`-separated index. Releases up to v0.3.0 use the old
layout, which xcl v0.13.0 and later no longer install.
