# cli-helper

Some helpers that I use in the terminal for which I did want to write too sophisticated bash foo.

## Tasks

- `decode-secret`: Decode K8s secret (also from yaml passed from stdin)
- `decode-jwt`: Decode JWT (also when passed from stdin)
- `battery-daemon`: Runs a battery daemon listening on dbus upower signals and emits battery notifications
- `reconcile`: Sets a `gardener.cloud/operation=reconcile` annotation on a resource
- `maintain`: Sets a `gardener.cloud/operation=maintain` annotation on a resource

## Battery Daemon

It can be started with systemd, too:

```bash
$ cat ~/.config/systemd/user/battery-daemon.service
[Unit]
Description=Battery Notification Daemon
After=graphical-session.target

[Service]
Type=simple
ExecStart=/usr/local/bin/ch battery-daemon --daemonize
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=graphical-session.target

$ systemctl --user start battery-daemon.service
```
