# SSH Tunnel

[![GitHub](https://img.shields.io/github/license/c3b2a7/ssh-tunnel)](https://github.com/c3b2a7/ssh-tunnel/blob/master/LICENSE)
![GitHub Workflow Status](https://img.shields.io/github/workflow/status/c3b2a7/ssh-tunnel/Build%20and%20test)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/c3b2a7/ssh-tunnel)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/c3b2a7/ssh-tunnel)

> A tunneling tool based on ssh protocol can be used for port forwarding. No dependency and out of the box.

## Features

- support `local`,`remote`,`dynamic` ssh port forwarding
- support `ssh-key`,`password` authentication method

## Usage

```shell
./sshtunnel-macos-arm64
Usage of ./sshtunnel-macos-arm64:
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