# crickChain
Go implementation of the crick proof-of-solution algorythm.


The problem is retrieved from the IPFS. Each block contains a problem reference and a possible solution. If the solution is valid, the difficulty decreases.

## Setup

```
go get github.com/joho/godotenv
go get github.com/davecgh/go-spew/spew
go get github.com/ipfs/go-log"
go get github.com/whyrusleeping/go-logging
go get github.com/libp2p/go-libp2p
go get github.com/libp2p/go-libp2p-crypto
go get github.com/libp2p/go-libp2p-host
go get github.com/libp2p/go-libp2p-net
go get github.com/libp2p/go-libp2p-peer
go get github.com/libp2p/go-libp2p-peerstore
go get github.com/multiformats/go-multiaddr
```

## Run
In a terminal:

```go run *.go -m true
```

Get output address and copy paste the command in another terminal. It looks something like this:

```
go run *.go -l 10001 -d /ip4/127.0.0.1/tcp/10000/ipfs/QmRrJF1WDt5rpTNSSfqmFkvHoo4JAV3ajoyU8PnEd7C1zT
```