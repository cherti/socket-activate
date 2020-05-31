# socket-activate

This is a little tool to add systemd socket activation to services that do not support it yet. This way services don't have be started at system boot, but are instead started on demand on first interaction (and possibly never if never used).

`socket-activate` also supports deactivating services again after a certain time of inactivity to save system resources.

## Usage

`socket-activate` can be used on its own, but it's most useful when used in conjunction with a systemd-socket-unit, obtaining systemd socket activation for arbitraty services (although currently only tcp-sockets are implemented, though).

### socket-activate itself
    Usage of ./socket-activate:
      -a string
            destination address (default "127.0.0.1:80")
      -m string
            mode, available: tcp (default "tcp")
      -t duration
            inactivity timeout after which to stop the unit again
      -u string
            corresponding unit (default "null.service")

### Usage example: Grafana

Deploy a unit `/etc/systemd/system/socket-activate-grafana.service`:

    [Unit]
    Description=Socket activation proxy for grafana.service
    Requires=socket-activate-grafana.socket
    
    [Service]
    ExecStart=/usr/local/bin/socket-activate -u grafana.service -a "127.0.0.1:3000" -t 15m
    NonBlocking=true

and a corresponding unit `/etc/systemd/system/socket-activate-grafana.socket`:

    [Unit]
    Description=Socket for grafana.service
    
    [Socket]
    ListenStream=127.0.0.1:1234
    NoDelay=true
    
    [Install]
    WantedBy=multi-user.target

You can now leave `grafana.service` disabled and stopped, it will automatically be activated once you access `127.0.0.1:1234` and proxy all requests to the Grafana instance behind.
If `-t` is specified, Grafana will be stopped again after the specified amount of time of no interaction (in this case 15min).

## How to get it

### Arch

[Install from AUR.](https://aur.archlinux.org/packages/socket-activate/)

### Build from source

Either

    go get -u github.com/cherti/typemute

and find the binary in `GOPATH/bin` (defaults to `~/go/bin`) or clone and

    go get -u github.com/sqp/pulseaudio  # get the dependency
    go build typemute.go  # and build locally
