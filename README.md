# xcl plugin template

A GitHub template for writing an [xcl](https://github.com/jumppad-labs/xcl)
plugin. It is a complete, working plugin with one resource, `notes "note"`,
which writes a note to a file. Unchanged, it builds, passes its unit and
end-to-end tests and applies its sample configuration, with nothing else
running. The same plugin can be registered inside an application or run as a
separate plugin program.

It follows xcl's
[standard plugin layout](https://github.com/jumppad-labs/xcl/blob/main/docs/plugin-layout.md),
so the plugin you build from it has the same shape as every other xcl plugin.

```hcl
notes "note" "greeting" {
  directory = "./out"
  name      = "greeting.txt"
  content   = "Hello from xcl"
}
```

You need Go (the version in `go.mod` or later) and `make`. Every command below
runs from the repository root.

## Start

Click **Use this template** on GitHub, or create your repository with the
GitHub CLI:

```shell
gh repo create <you>/xcl-plugin-<name> --template jumppad-labs/xcl-plugin-template --public --clone
cd xcl-plugin-<name>
```

Name the repository after your plugin, for example `xcl-plugin-files`: the
GitHub registry installs a plugin by its repository name, and the release
assets are named after it.

## Make it yours

The template builds and passes as it is, so make these renames one at a time
and run `make test` after each:

1. **Module path.** Change `module` in `go.mod` to your repository, for
   example `github.com/<you>/xcl-plugin-<name>`, then replace
   `github.com/jumppad-labs/xcl-plugin-template` with it in every import
   and in `.mockery.yml`.
2. **Plugin name.** Set `PLUGIN` at the top of the `Makefile` to your
   repository's name. It names the binary and every release asset.
3. **Root package and program.** Rename the package in `plugin.go` from
   `notes` to your plugin's domain, and the directory `cmd/notes` to match.
   Update `./cmd/notes` in the `Makefile` and the `e2e` tests.
4. **Block types.** Replace the note with your own resources, as
   [Add your own resource](#add-your-own-resource) shows, and change the
   type registered in `plugin.go`.

## Layout

```
plugin.go                  the plugin type: Init builds the files client once and registers notes "note"
cmd/notes/main.go          serves the plugin as a separate program
entities/note.go           the notes "note" block type, and nothing else
providers/note.go          the note provider: create, read, decide, update, destroy
providers/note_test.go     unit tests of the provider against a strict double
client/files/files.go      the Files interface and its local-disk implementation
client/files/mocks/        the Mockery-generated double of Files
examples/basic/main.xcl    the sample configuration
e2e/                       end-to-end tests: apply, plan with no changes, destroy
.mockery.yml               which interfaces get a double
Makefile                   build, test, generate, inprocess, external
```

The rules behind it:

- `entities/` holds block types only. It imports nothing else from the
  plugin, so an application that only reads your resources from state
  imports `entities` alone. `e2e/stateonly` proves it: it builds with only
  xcl and `entities`, and reads the applied note back from state.
- `client/` is the only place a backend is touched (here, the local disk).
  Its interfaces hold only the calls the providers make, and its packages
  never import providers or entities.
- `providers/` builds each provider with a constructor that takes the
  clients it uses, so the tests pass it a strict double.
- Every file lists its exported types, vars and functions first and its
  unexported helpers last. A provider's methods follow the lifecycle order:
  `Init`, `Create`, `Read`, `Changed`, `Update`, `Destroy`, `Functions`.

## Build

```shell
make build
```

Builds every package and the plugin program, `build/<PLUGIN>`.

## Test

```shell
make test
```

Runs the unit tests and the end-to-end tests, with the race detector. Nothing
else needs to be running: the plugin only writes files to temporary
directories.

## Regenerate the test double

```shell
make generate
```

Regenerates `client/files/mocks` from the interfaces listed in `.mockery.yml`.
Run it whenever you change a client interface. On an unchanged template it
produces no changes.

## Use it in-process

```shell
make inprocess
```

Applies `examples/basic`, plans it again with no changes and destroys it,
with the plugin imported and registered inside the application. This is how
an application registers it, from `e2e/main_test.go`:

```go
local := registry.NewLocal()
local.RegisterPlugin(&notes.Plugin{})

c, err := xcl.NewConfig(
	xcl.WithRegistry(local),
	xcl.WithStatePath(stateDir),
	xcl.WithVariables(map[string]any{"directory": outDir}),
)
```

## Use it as a separate program

```shell
make external
```

Does the same with the plugin built from `cmd/notes` and run as a separate
plugin program, which xcl starts and talks to over gRPC. The application
registers the binary instead of the type:

```go
local := registry.NewLocal()
local.RegisterExternalPlugin(pluginBinary)
```

`pluginBinary` is the path of the built program, for example
`build/xcl-plugin-template`.

## The change decision

When a resource's settings change, the provider's `Changed` decides whether it
is updated in place or replaced. The template makes that decision explicit, at
the top of `providers/note.go`:

```go
var ReplaceSettings = []entity.Path{
	entity.Path{}.Attribute("directory"),
	entity.Path{}.Attribute("name"),
}
```

`Changed` reads in a fixed order:

1. a change `Within` any of `ReplaceSettings` answers **replace**: moving the
   file means removing the old one and writing a new one;
2. a dependency that is being replaced answers **replace**. A resource built
   on another type narrows this to that type's dependencies;
3. any other change answers **update**;
4. with no changes, it defers to `DefaultChanged`.

To change what needs a replace, edit the list, not the logic. The unit tests
have one function per rule, and `e2e/changes_test.go` shows the same outcome
in a real plan.

## Updating from what you are told

`Update` receives the settings that changed, each with its `Before` and
`After` value, and the dependencies that changed. It acts on those alone and
never looks the resource up to find out what it was:

- a `content` change rewrites the file and keeps the new checksum as a
  computed value, so the next decision needs no lookup;
- a `mode` change on its own only changes the file's mode.

The unit tests drive `Update` against a strict double that expects only the
calls those changes need, so a lookup such as `Stat` fails the test.

## Add your own resource

To add a resource, for example a `notes "folder"`:

1. **Entity.** Add `entities/folder.go` with the block type: a struct
   embedding `types.ResourceBase` with `xcl:",remain"`, its settings and its
   computed values (`xcl:"<name>,optional,computed"`). Use plain field types,
   and import nothing from the rest of the plugin.
2. **Client.** If it needs a backend the plugin does not have yet, add a
   package under `client/<backend>/` with a narrow interface holding only the
   calls the provider will make, and its real implementation. Return the
   package's own types, never entities.
3. **Double.** Add the interface under `packages` in `.mockery.yml` and run
   `make generate`.
4. **Provider.** Add `providers/folder.go`: its replace-only settings list,
   the provider type, its interface assertion and a constructor taking its
   clients, then `Init`, `Create`, `Read`, `Changed`, `Update`, `Destroy` and
   `Functions` in that order, and unexported helpers last. Copy the shape of
   `providers/note.go`.
5. **Unit tests.** Add `providers/folder_test.go`, building the provider
   with a fresh double in each test. Write one test function per rule, and
   keep positive and negative cases in separate functions.
6. **Register it.** In `plugin.go`, build any new client once and call
   `plugins.RegisterResourceProvider` for the new type.
7. **Sample and end-to-end test.** Add the block to a configuration under
   `examples/` and an `e2e` test that applies it, plans with no changes and
   destroys it.

## Checks

```shell
make lint
```

Runs `go vet`, a `gofmt` check and staticcheck. GitHub Actions runs
`make build`, `make lint` and `make test` on every push and pull request
(`.github/workflows/ci.yml`), and once a week runs the suite against the
latest xcl release (`.github/workflows/current.yml`, which can also be started
by hand), so a release that breaks the plugin shows up on its own.

## Signing key

Releases are signed, so applications can check that a plugin came from you.
Make an OpenPGP key once, Ed25519 or RSA 4096 (both verify with the GitHub
registry):

```shell
gpg --quick-generate-key "Your Name (xcl plugin releases) <you@example.com>" ed25519 sign never
```

Then give it to the release workflow as two repository secrets, and publish
the public key:

1. **`GPG_PRIVATE_KEY`**: the armoured private key,
   `gpg --armor --export-secret-keys <fingerprint>`.
2. **`GPG_PASSPHRASE`**: its passphrase.

   ```shell
   gpg --armor --export-secret-keys <fingerprint> | gh secret set GPG_PRIVATE_KEY
   gh secret set GPG_PASSPHRASE
   ```

3. **The public key**: export it with `gpg --armor --export <fingerprint>`
   and publish it somewhere stable you control, such as your website, along
   with its fingerprint. Applications that install your plugin trust it.

Without `GPG_PRIVATE_KEY` the release workflow fails before building, so an
unsigned release is never published.

## Release

```shell
make release VERSION=v0.1.0
```

Tags the current commit and pushes the tag. The release workflow
(`.github/workflows/release.yml`) runs the tests, builds the archives with
`make dist`, imports the signing key, signs the checksums file and publishes
the release with `gh`. With `PLUGIN` set to your repository's name, the
release holds:

- `<PLUGIN>_0.1.0_<os>_<arch>.tar.gz` for linux and darwin, and
  `<PLUGIN>_0.1.0_windows_<arch>.zip`, on amd64 and arm64, each with the
  binary at its root, beside `README.md` and `LICENSE`;
- `<PLUGIN>_0.1.0_checksums.txt`, the SHA-256 of every archive in
  `sha256sum` format;
- `<PLUGIN>_0.1.0_checksums.txt.sig`, an armoured detached signature over the
  checksums file.

To build the same archives locally, unsigned:

```shell
make dist VERSION=v0.0.0-snapshot
```

CI runs this on every change, so broken release packaging fails on the change
that broke it. Release packaging needs only go, tar, zip, sha256sum, gpg and
gh, all already on GitHub's `ubuntu-latest` runner, so it adds no tool to pin.

## Install

An application installs a released plugin with xcl's GitHub registry, which
downloads the build for its platform, checks it against the checksums and,
with a trusted key, the signature, and runs it as a separate program:

```go
gh := registry.NewGitHub(registry.GitHubTrustedKeys(publicKey))
gh.RegisterPlugin("<you>/xcl-plugin-<name>", "v0.1.0")

c, err := xcl.NewConfig(xcl.WithRegistry(gh))
```

`publicKey` is the armoured public key you published. See
[Installing plugins from GitHub](https://github.com/jumppad-labs/xcl#installing-plugins-from-github)
for pinning, private repositories and the cache.

## Verifying releases

The template's own releases are signed with the shared jumppad-labs release
key, published at
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
gh.RegisterPlugin("jumppad-labs/xcl-plugin-template", "v0.1.0")
```

## Tools and why

The plugin's only Go dependencies are xcl itself and
[testify](https://github.com/stretchr/testify), which provides `require` and
the mock runtime the generated double uses. The tools are pinned in the
`Makefile` and run with `go run`, so there is nothing to install:

- **[Mockery](https://github.com/vektra/mockery) v3.8.0** generates the
  strict test doubles of the client interfaces.
- **[staticcheck](https://staticcheck.dev) v0.8.1** runs the static checks
  beyond `go vet`, in `make lint` and CI. It needs Go 1.26 or later, which
  `go run` fetches for it when `GOTOOLCHAIN` is `auto`, the default.
