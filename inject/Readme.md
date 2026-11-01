# Dependency injection

Package `inject` provides dependency injection inspired by
[Guice](https://github.com/google/guice). A `Module` groups bindings, and an
`Injector` assigns matching instances to exported fields on struct pointers.

## Bindings and tags

```go
module := inject.NewModule()
module.Bind(&DefaultStore{})
module.BindWithName("primary", &SQLStore{})

injector := inject.NewInjector([]*inject.Module{module})
```

Assuming both store types implement `Store`, a target can request them by type
and optional name:

```go
type Handler struct {
	Default Store   `inject:"_"`
	Primary Store   `inject:"primary"`
	Backup  Store   `inject:"backup,optional"`
	All     []Store `inject:"_"`
}

handler := &Handler{}
injector.RequireInject(handler)
```

- `_` selects an unnamed binding assignable to the field type.
- A name selects a matching named binding.
- `optional` permits a missing binding. A missing optional scalar field keeps
  its existing value; an optional slice becomes an empty slice if nothing matches.
- A slice receives all assignable bindings, including named bindings. Named
  slice injection is not supported.
- Fields without an injection tag are left unchanged. Unexported fields are
  skipped. Initialized embedded structs and struct pointers are also traversed.

Missing required bindings and ambiguous scalar bindings panic. Configure the
injector during application startup so binding problems surface before requests
are served. Injection targets must be non-nil struct pointers.

`NewInjector` immediately injects the bound module instances. Dependencies for
objects created separately are injected with `RequireInject`.

## Providers

Use `BindWithProvider` with a `Provider`, or `BindWithProviderFunc` with a
constructor function:

```go
module.BindWithProviderFunc("primary", func() interface{} {
	return &SQLStore{}
})
```

Providers execute when the binding is added. The resulting instance is reused;
providers are not invoked on every lookup. Returning nil causes a panic.

## Override tags

Override tags without changing the target's struct definition:

```go
injector.RequireInjectWithOverrideTags(handler, map[string]string{
	"Default": "primary",
})
```

`BindWithNameOverrideTags` applies overrides when a module instance is injected.
An instance implementing `Injected` can return true from `IsInjected()` to skip
subsequent injection when no override tags are supplied.

## Lookups

Use `GetInstance` with a `reflect.Type`, or use a single-field struct prototype
to express the requested type:

```go
store := injector.GetInstanceByPrototype("primary", struct{ Store Store }{})
stores := injector.GetInstancesByPrototype(struct{ Store Store }{})
```

Single-instance lookups return nil if no binding matches and panic if multiple
bindings match. Collection lookups include both named and unnamed bindings.

## Initialization and service lifecycles

```go
if err := injector.Initialize(); err != nil {
	return err
}
if err := injector.Start(); err != nil {
	return err
}
defer injector.Stop()
```

`Initialize` calls `Init() error` on bound instances implementing
`common.Initable`. Bound `common.Service` instances also receive lifecycle state
updates. Embed `common.BaseService` to supply the base implementation, including
its internal state-management method.

`Start` and `Stop` order services by `GetStartOrder()` and `GetStopOrder()`.
`BaseService` uses `Order` for startup and `-Order` for shutdown.

## YAML configuration

`SetupInjector` and `SetupInjectorWithLoader` load configuration, call its
`Parse` method, bind it, inject dependencies, and initialize bound instances.
They return an initialized injector; call `Start` when the application is ready.

Configuration is loaded from `conf`, or the `conf_root` environment variable:

1. Inline `addonConfig`, if supplied.
2. `common.yaml`, if present.
3. `conf_<env>.yaml`, or `conf.yaml` when `env` is empty, if present.

The sources are concatenated and decoded as one YAML document. Avoid duplicate
mapping keys across sources; this is not a recursive configuration merge.

A configuration implementing `ConfigModuler` can return extra bindings through
`ConfModule() (*Module, error)`. Supply a `common.ConfigLoader` to
`SetupInjectorWithLoader` to load configuration from another source.
