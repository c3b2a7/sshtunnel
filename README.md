# SSH Tunnel

[![GitHub](https://img.shields.io/github/license/c3b2a7/sshtunnel)](https://github.com/c3b2a7/sshtunnel/blob/master/LICENSE)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/c3b2a7/sshtunnel/build.yml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/c3b2a7/sshtunnel)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/c3b2a7/sshtunnel)

> A tunneling tool based on ssh protocol can be used for port forwarding. No dependency and out of the box.

## Features

- support `local`,`remote`,`dynamic` ssh port forwarding
- support `ssh-key`,`password` authentication method

## Installation

### Using curl/sh:

```shell
curl https://raw.githubusercontent.com/c3b2a7/sshtunnel/master/scripts/get-sshtunnel.sh | sh
```

See the help output for more options:

```shell
curl https://raw.githubusercontent.com/c3b2a7/sshtunnel/master/scripts/get-sshtunnel.sh | sh -s -- -h
```

### From source:

```shell
go install github.com/c3b2a7/sshtunnel@latest
```

### Manual

You can also download and extract the latest release from
[https://github.com/c3b2a7/sshtunnel/releases](https://github.com/c3b2a7/sshtunnel/releases)

## Usage

```shell
./sshtunnel
Usage of ./sshtunnel:
  -config string
    	config file
  -v	show version information
  -verbose
    	verbose mode
```

## Quick Start

At first, you need write a configuration like this:

```json
{
  "target": "host:port",
  "username": "username",
  "private-key": "location of ssh private key",
  "passphrase": "private-key passphrase or password of username",
  "tunnels": [
    {
      "local": "127.0.0.1:13306",
      "remote": "172.16.0.14:3306",
      "mode": "local"
    },
    {
      "local": "127.0.0.1:8080",
      "remote": "0.0.0.0:18080",
      "mode": "remote"
    },
    {
      "local": "127.0.0.1:1080",
      "mode": "dynamic"
    }
  ]
}
```

and then, use the following command to start ssh tunnel:

```shell
./sshtunnel -config /path/to/config -verbose
```

after the tunnel is established:

connect to the remote MySQL service like connecting to the local：

```shell
mysql -h 127.0.0.1 -P 13306 -u root -p # in local
```

connect to local service in remote:

```shell
nc -l 8080 # in local
nc localhost 18080 # in remote
```

connect to dynamic addr using socks5 protocol via remote server:

```shell
curl -x socks5://localhost:1080 ip.sb # in local
```

# LICENSE

MIT