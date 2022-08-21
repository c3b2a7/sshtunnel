# SSH Tunnel

> A tunneling tool based on ssh protocol can be used for port forwarding. No dependency and out of the box.

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
  "ssh-key": "location of ssh private key",
  "passphrase": "password",
  "tunnels": [
    {
      "local": "127.0.0.1:13306",
      "remote": "172.16.0.14:3306"
    },
    {
      "local": "127.0.0.1:8080",
      "remote": "172.16.0.14:18080"
    }
  ]
}
```

and then, use the following command to start ssh tunnel:

```shell
./sshtunnel -config /path/to/config -verbose
```

after the tunnel is established, connect to the remote MySQL service like connecting to the local：

```shell
mysql -h 127.0.0.1 -P 13306 -u root -p
```

# LICENSE

MIT