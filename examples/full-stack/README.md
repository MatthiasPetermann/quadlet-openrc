# Full-stack reference example

Copy the Quadlet files and drop-in directory to `/etc/containers/openrc/`, then run:

```sh
quadlet-openrc check
quadlet-openrc lint
quadlet-openrc generate --clean
rc-update add quadlet-network-frontend default
rc-update add quadlet-volume-webdata default
rc-update add quadlet-image-nginx default
rc-update add web default
rc-service web start
```

Generated service graph:

```text
web
├── need quadlet-image-nginx
├── need quadlet-network-frontend
└── need quadlet-volume-webdata
```

Podman object names remain independent from OpenRC service names:

```text
frontend.network -> OpenRC quadlet-network-frontend -> Podman network frontend
webdata.volume   -> OpenRC quadlet-volume-webdata   -> Podman volume webdata
nginx.image      -> OpenRC quadlet-image-nginx      -> image localhost/systemd-nginx
web.container    -> OpenRC web                      -> container web
```
