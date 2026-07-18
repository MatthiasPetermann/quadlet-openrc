# Reference example

This example contains only declarative Quadlet files (`.container`, `.network`, `.volume`, `.image`, plus drop-ins).

## Set up on Alpine

1. Copy declarative files to `/etc/containers/openrc/`.
2. Run the generator to produce services.
3. Add services to the desired runlevel.
4. Start the stack and verify status.

```sh
quadlet-openrc check
quadlet-openrc lint
quadlet-openrc generate --clean
rc-update add quadlet-network-frontend default
rc-update add quadlet-volume-webdata default
rc-update add quadlet-image-nginx default
rc-update add web default
rc-service web start
rc-service web status
```

## Operations and troubleshooting

Useful OpenRC commands:

```sh
rc-service quadlet-network-frontend status
rc-service quadlet-volume-webdata status
rc-service quadlet-image-nginx status
rc-service web restart
rc-status -a
```

Logs are available in `/var/log/quadlet/` for generated container services, and via Podman for container runtime logs:

```sh
ls -l /var/log/quadlet/
podman ps --format 'table {{.Names}}\t{{.Status}}'
podman logs web
podman logs --since 10m web
```

Podman object names remain independent from OpenRC service names:

```text
frontend.network -> OpenRC quadlet-network-frontend -> Podman network frontend
webdata.volume   -> OpenRC quadlet-volume-webdata   -> Podman volume webdata
nginx.image      -> OpenRC quadlet-image-nginx      -> image localhost/systemd-nginx
web.container    -> OpenRC web                      -> container web
```
