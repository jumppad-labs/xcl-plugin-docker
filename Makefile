# Builds, tests, regenerates the doubles of and releases the Docker plugin.
# Run every target from the repository root.

# PLUGIN is the plugin's name: the name of its binary and of every release
# asset. It must match the name of your repository, because the GitHub
# registry installs a plugin by its repository name. Change it when you make
# the template your own.
PLUGIN := xcl-plugin-docker

BUILD_DIR := build
DIST_DIR := dist

# PLATFORMS are the operating system and architecture pairs a release is
# built for
PLATFORMS := linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64 windows_arm64

# VERSION is the release version, a tag such as v1.2.0 or v1.2.0-rc.1. The
# dist and release targets require it, for example make dist VERSION=v0.1.0.
VERSION ?=

# DIST_VERSION is VERSION without its leading v, as release asset names use it
DIST_VERSION = $(VERSION:v%=%)

# MOCKERY is the pinned Mockery release that generates the client doubles,
# run through go run so it needs no install
MOCKERY := go run github.com/vektra/mockery/v3@v3.8.0

# STATICCHECK is the pinned staticcheck release the lint target runs, through
# go run so it needs no install. It needs Go 1.26 or later, which go run
# fetches when GOTOOLCHAIN is auto, the default
STATICCHECK := go run honnef.co/go/tools/cmd/staticcheck@v0.8.1

.PHONY: build test lint generate e2e dist release check-version clean

# Build every package, and the plugin program as build/$(PLUGIN)
build:
	go build ./...
	go build -o $(BUILD_DIR)/$(PLUGIN) ./cmd/docker

# Run the unit and end-to-end tests. The tests that need a Docker engine skip
# when none is reachable.
test:
	go test -race ./...

# Run the static checks CI runs: go vet, a gofmt check and staticcheck
lint:
	go vet ./...
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "not gofmt formatted:"; echo "$$unformatted"; exit 1; fi
	$(STATICCHECK) ./...

# Regenerate the double of each client interface, client/*/mocks
generate:
	$(MOCKERY)

# Apply the sample, plan it again with no changes and destroy it against a
# real Docker engine, with the plugin built and run as a separate plugin
# program. The tests skip when no engine is reachable.
e2e:
	go test -v -count=1 ./e2e

# Build the release into dist/, in the layout the GitHub registry installs:
# for each platform <PLUGIN>-<version>-<os>-<arch>.zip holding the binary,
# README, LICENSE and the build's xcl-plugin.xcl manifest at its root, and
# <PLUGIN>-<version>-index.xcl listing every zip with its sha256 checksum.
# The names are separated with -, like the repository's name; the registry
# also accepts _ used throughout.
# It uses only go, zip and sha256sum, and signs nothing: the release workflow
# signs the index.
dist: check-version
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	@set -e; \
	index="$(DIST_DIR)/$(PLUGIN)-$(DIST_VERSION)-index.xcl"; \
	platforms=""; \
	for platform in $(PLATFORMS); do \
		os=$${platform%_*}; arch=$${platform#*_}; \
		ext=""; if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		stage="$(DIST_DIR)/stage/$$platform"; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -o "$$stage/$(PLUGIN)$$ext" ./cmd/docker; \
		cp README.md LICENSE "$$stage/"; \
		printf 'plugin {\n  name    = "%s"\n  version = "%s"\n}\n\nexecutable {\n  protocol = [1]\n  command  = ["%s"]\n\n  platform "%s" "%s" {}\n}\n' \
			"$(PLUGIN)" "$(DIST_VERSION)" "$(PLUGIN)$$ext" "$$os" "$$arch" > "$$stage/xcl-plugin.xcl"; \
		archive="$(PLUGIN)-$(DIST_VERSION)-$$os-$$arch.zip"; \
		(cd "$$stage" && zip -q -X "../../$$archive" *); \
		checksum=$$(sha256sum "$(DIST_DIR)/$$archive" | cut -d' ' -f1); \
		platforms="$$platforms$$(printf '\n  platform "%s" "%s" {\n    asset    = "%s"\n    checksum = "sha256:%s"\n  }\n' "$$os" "$$arch" "$$archive" "$$checksum")\n"; \
	done; \
	printf 'plugin {\n  name    = "%s"\n  version = "%s"\n}\n\nexecutable {\n  protocol = [1]\n' "$(PLUGIN)" "$(DIST_VERSION)" > "$$index"; \
	printf "$$platforms" >> "$$index"; \
	printf '}\n' >> "$$index"
	rm -rf $(DIST_DIR)/stage
	@ls -1 $(DIST_DIR)

# Release VERSION: tag the current commit and push the tag. The release
# workflow then builds, signs and publishes the release.
release: check-version
	git tag $(VERSION)
	git push origin $(VERSION)

# Fail unless VERSION is a release version, v<major>.<minor>.<patch> with an
# optional pre-release suffix
check-version:
	@echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || \
		{ echo "usage: make $(MAKECMDGOALS) VERSION=v<major>.<minor>.<patch>[-<pre-release>], for example VERSION=v0.1.0"; exit 1; }

# Remove the built binary and the release archives
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)
