# quadlet-openrc 0.3

Quadlet is the declarative container-unit format introduced in the systemd ecosystem and heavily adopted in Red Hat and derivative distributions. It offers a pragmatic abstraction for describing Podman workloads as small, composable unit files.

For single-node appliances, Alpine Linux is often a strong operating-system choice because it is compact and has a reduced attack surface. This project brings a carefully selected subset of Quadlet to Alpine as an OpenRC generator, so the model fits natively into an OpenRC-based host without adding a systemd compatibility layer.

## Features

- `.container`, `.network`, `.volume`, and `.image`
- systemd-style drop-ins: `name.type.d/*.conf`
- `Requires`, `Wants`, `After`, and `Before`
- implicit dependencies from `Image=.image`, `Network=.network`, and `Volume=.volume`
- missing dependency and cycle detection
- atomic generation and safe stale-file cleanup
- OpenRC supervision through `supervise-daemon`
- rootful hardening options without requiring SELinux
- security lint mode
- complete runnable reference example

## Build on Alpine

```sh
apk add --no-cache go podman openrc iptables ip6tables
make test
make build
install -m 0755 bin/quadlet-openrc /usr/local/sbin/quadlet-openrc
```

## Stable CLI

```sh
quadlet-openrc check
quadlet-openrc lint
quadlet-openrc generate --dry-run
quadlet-openrc generate --clean
quadlet-openrc version
```

## Stable naming

```text
frontend.network -> quadlet-network-frontend -> Podman network frontend
webdata.volume   -> quadlet-volume-webdata   -> Podman volume webdata
nginx.image      -> quadlet-image-nginx      -> tagged Podman image
web.container    -> web                      -> Podman container web
```

Quadlet references always use filenames:

```ini
[Container]
Image=nginx.image
Network=frontend.network
Volume=webdata.volume:/data
```

## Hardened rootful operation

```ini
[Container]
UserNS=auto
DropCapability=ALL
NoNewPrivileges=true
ReadOnly=true
Tmpfs=/tmp:rw,noexec,nosuid,size=64m
Tmpfs=/run:rw,noexec,nosuid,size=16m
PidsLimit=256
Memory=512m
CPUs=1
```

These controls work without SELinux. AppArmor or another MAC system remains an optional additional layer.

## Supported container keys

`Image`, `ContainerName`, `Exec`, `Pull`, `Environment`, `EnvironmentFile`, `Volume`, `Network`, `PublishPort`, `ExposeHostPort`, `Label`, `User`, `UserNS`, `WorkingDir`, `HostName`, `ReadOnly`, `RunInit`, `Privileged`, `AddCapability`, `DropCapability`, `NoNewPrivileges`, `Tmpfs`, `PidsLimit`, `Memory`, `CPUs`, `Secret`, `Device`, `HealthCmd`, `HealthInterval`, `HealthTimeout`, `HealthRetries`, and `PodmanArgs`.

Supported service keys are `Restart`, `RestartSec`, and `TimeoutStopSec`.

## Deliberate non-goals

- socket activation and timers
- `sd_notify` fidelity
- systemd credentials
- full `PartOf` restart propagation
- arbitrary systemd sandbox directives
- Kubernetes orchestration
- rootless per-user service management

These would require a second service-manager layer and would undermine the OpenRC-native design.

See `STABLE-API.md` and `examples/full-stack/`.
