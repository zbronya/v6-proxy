# v6-proxy
A random IPv6 proxy service based on Go

## Installation

```bash
curl -sSL https://github.com/zbronya/v6-proxy/raw/master/install.sh | sudo bash
```

## Usage
```bash
sudo v6-proxy --cidr=2001:db8::/32
```
Please replace the --cidr parameter with your IPv6 address range.


test
```bash
while true; do curl -x http://127.0.0.1:33300 -s https://api.ip.sb/ip -A Mozilla; done
```


## Parameters
- `--cidr` ipv6 address segment, required
- `--port` proxy server port, default 33300
- `--username` proxy server username, default empty
- `--password` proxy server password, default empty
- `--bind` proxy server bind address, default 127.0.0.1
- `--auto-route` auto add route to the system, default true **need root permission**
- `--auto-forwarding` auto add forwarding to the system, default true **need root permission**

## what's next





