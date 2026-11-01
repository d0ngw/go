# Go Toolkit

A Go toolkit for dependency injection, service lifecycles, YAML configuration,
HTTP services, Redis caching, and MySQL data access.

## Requirements

- Go 1.26 or later. The recommended toolchain is Go 1.26.10.
- Redis for the Redis cache, counter, and list packages.
- MySQL for database operations and persisted counters and lists.

## Installation

```sh
go get github.com/d0ngw/go
```

Import the packages you need; there is no package at the module root.

## Packages

| Package | Purpose |
| --- | --- |
| [`common`](common) | Configuration, logging, service lifecycles, validation, collections, and utility functions |
| [`common/perm`](common/perm) | Resources, roles, principals, and permission checks |
| [`inject`](inject/Readme.md) | Dependency injection inspired by [Guice](https://github.com/google/guice) |
| [`http`](http) | HTTP services, controllers, middleware, and client helpers |
| [`cache`](cache) | Redis clients, connection pools, pipelines, and MessagePack encoding |
| [`cache/counter`](cache/counter) | Redis counters and database persistence |
| [`cache/list`](cache/list) | Redis-backed lists with database persistence |

## Dependency injection

Bind concrete instances to a module, then inject them into exported struct fields.
Use `inject:"_"` for an unnamed binding or a name such as `inject:"primary"` for a
named binding.

```go
package main

import (
	"fmt"

	"github.com/d0ngw/go/inject"
)

type Greeting interface {
	Message() string
}

type EnglishGreeting struct{}

func (*EnglishGreeting) Message() string { return "Hello from Go Toolkit!" }

type Application struct {
	Greeting Greeting `inject:"_"`
}

func main() {
	module := inject.NewModule()
	module.Bind(&EnglishGreeting{})

	injector := inject.NewInjector([]*inject.Module{module})
	app := &Application{}
	injector.RequireInject(app)

	fmt.Println(app.Greeting.Message())
}
```

See the [dependency injection guide](inject/Readme.md) for named bindings,
providers, configuration loading, and service lifecycles.

## Configuration and services

`inject.SetupInjector(config, addonConfig, env, modules...)` loads YAML from the
`conf` directory, or the directory specified by the `conf_root` environment
variable. It combines the inline `addonConfig` with `common.yaml` and
`conf_<env>.yaml` (`conf.yaml` when `env` is empty), skipping missing files. At
least one source must contain configuration. It then calls `config.Parse()`,
binds the configuration, injects dependencies, and initializes bound instances.

Embed `common.BaseService` when implementing `common.Service`. Call
`injector.Initialize()` after constructing an injector directly, followed by
`injector.Start()` and `injector.Stop()` to manage service lifecycles. Services
start in ascending `GetStartOrder()` order and stop in ascending
`GetStopOrder()` order. `BaseService` defaults to stopping in reverse startup
order.

## Development

The default suite needs no Redis or MySQL services. HTTP tests use dynamically
allocated localhost ports.

```sh
go mod download
go build ./...
go vet ./...
go test -race ./...
```

To run integration tests with disposable MySQL 8.4 and Redis 7 services:

```sh
docker compose up -d --wait
go test -tags integration -race -count=1 ./...
docker compose down
```

The Compose services bind only to localhost. MySQL uses temporary storage and
Redis persistence is disabled. Tests create and drop unique databases, use
isolated Redis key prefixes, and load embedded Lua scripts and checked-in SQL
fixtures. Packages can run concurrently. The MySQL test account needs permission
to create and drop test databases.

For other disposable service instances, configure:

| Variable | Default | Purpose |
| --- | --- | --- |
| `MYSQL_ADDR` | `127.0.0.1:33306` | MySQL host and port |
| `MYSQL_USER` | `root` | Test database user |
| `MYSQL_PASSWORD` | `123456` | Test database password |
| `MYSQL_DATABASE` | `go_toolkit_test` | Prefix for generated test database names |
| `REDIS_ADDR` | `127.0.0.1:36379` | Redis host and port |
| `REDIS_PASSWORD` | empty | Redis authentication password |

`MYSQL_DATABASE` must contain 1 to 30 letters, digits, or underscores. Tests
operate on Redis database zero using generated key prefixes, not a database-wide
flush. Use dedicated services rather than application databases.

GitHub Actions runs unit tests on Go 1.26 and the current stable Go release,
checks formatting and modules, runs the integration suite with MySQL and Redis,
fuzzes the AES decoder, and scans for known vulnerabilities. Dependabot checks Go
modules and Actions dependencies weekly.

## Maintenance

This refresh raises the minimum Go version to 1.26 and updates dependencies to
their current compatible releases. `github.com/json-iterator/go` and
`gopkg.in/yaml.v3` remain at their latest releases on their existing module paths.
A YAML module-path migration needs a separate compatibility review because
applications can implement custom YAML unmarshaling methods using package types.

HTTP initialization now returns an error and the default AES ciphertext format
now uses authenticated encryption.

Persistent counters use the original `counter.Persist` interface: `Load`, `Store`,
and `Del`. Redis scripts make counter updates atomic across clients. Cache-miss
loads, deletion, and background saves run without a local mutex or distributed
coordination. Persistence backends must support concurrent calls. Applications
must coordinate these operations when ordering between Redis and persistence is
required: concurrent saves can overwrite newer persisted snapshots, and a save
racing deletion can recreate persisted data. Redis generation and version checks
protect acknowledgement and eviction, but do not make Redis and persistence a
single transaction. Treat Redis as the live counter state: changes are durable
only after a successful background save.

Counter fields must be non-empty and must not start with `_`, which is reserved
for internal metadata. Use `counter.DefaultScripts()` for the built-in scripts.
Custom update scripts must accept the `__g` generation argument; synchronization
and eviction scripts must check the generation supplied as their final argument.
Upgrade all writers and scanners together.
