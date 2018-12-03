package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"log"
	"fmt"
	mrand "math/rand"
	"strings"
	"time"

	//"github.com/davecgh/go-spew/spew"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

func makeBasicHost(listenPort int, randseed int64) (host.Host, error) {
	var r io.Reader

	//if randseed is not provided (=0) generate a random ID, else generate it deterministically
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}
	
	//generate key pair for this host to obtain valid host ID
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", listenPort)),
		libp2p.Identity(priv),
	}
	
	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	//host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))
	
	addrs := basicHost.Addrs()
	var addr ma.Multiaddr
	// select the address starting with "ip4"
	for _, i := range addrs {
		if strings.HasPrefix(i.String(), "/ip4") {
			addr = i
			break
		}
	}
	fullAddr := addr.Encapsulate(hostAddr)
	fmt.Printf("go run *.go -l %d -d %s\n", listenPort+1, fullAddr)
	
	return basicHost, nil
}

func handleStream(s net.Stream) {
	// log.Println("New stream connected")
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go broadcastData(rw)
	go receiveBroadcastData(rw)
	
}

func receiveBroadcastData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {

			data := BroadcastData{}
			if err := json.Unmarshal([]byte(str), &data); err != nil {
				log.Fatal(err)
			}

			Mutex.Lock()
			chain := data.Blockchain
			utxos := data.UTXOs
			if len(chain) > len(Blockchain) {
				Blockchain = chain
			}

			for k, v := range utxos {
			    UTXOs[k] = v
			}
			UTXOs = filterUTXOs(UTXOs)
			Mutex.Unlock()
		}
			
	}
}

func broadcastData(rw *bufio.ReadWriter) {
	//broadcast your blockchain version every BROADCAST_INTERVAL seconds.
	//We need to broadcast also unverified utxos
	for {
		time.Sleep(BROADCAST_INTERVAL)
		
		Mutex.Lock()
		UTXOs = filterUTXOs(UTXOs) //add this just to be sure that i'm not broadcasting invalid utxs
		Data = BroadcastData{Blockchain, UTXOs}
		bytes, err := json.Marshal(Data)
		if err != nil {
			log.Println(err)
		}
		Mutex.Unlock()

		Mutex.Lock()
		rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
		rw.Flush()
		Mutex.Unlock()

	}
}

func createListener(host host.Host) {
	log.Println("listening for connections")
	// Set a stream handler on host A. /p2p/1.0.0 is
	// a user-defined protocol name.
	host.SetStreamHandler("/p2p/1.0.0", handleStream)
}

func connectToPeer(host host.Host, target string) {
	host.SetStreamHandler("/p2p/1.0.0", handleStream)

	// The following code extracts target's peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(target)
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatalln(err)
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(
	fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

	// We have a peer ID and a targetAddr so we add it to the peerstore
	// so LibP2P knows how to contact it
	host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	log.Println("opening stream")
	// make a new stream from host B to host A
	// it should be handled on host A by the handler we set above because
	// we use the same /p2p/1.0.0 protocol
	s, err := host.NewStream(context.Background(), peerid, "/p2p/1.0.0")
	if err != nil {
		log.Fatalln(err)
	}
	handleStream(s)
}