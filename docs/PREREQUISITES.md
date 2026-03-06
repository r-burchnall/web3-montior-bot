# Prerequisites

## Go 1.26+

Required to build the fcsc-agent binary.

### Install on Ubuntu/Debian (amd64)

```bash
curl -sLO https://go.dev/dl/go1.26.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.1.linux-amd64.tar.gz
rm go1.26.1.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

### Install on Ubuntu/Debian (arm64)

```bash
curl -sLO https://go.dev/dl/go1.26.1.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.26.1.linux-arm64.tar.gz
rm go1.26.1.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

### Verify

```bash
$ go version
go version go1.26.1 linux/amd64
```
