package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"fmt"
	mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"sync"
	"github.com/joho/godotenv"
	"github.com/davecgh/go-spew/spew"
	golog "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	gologging "github.com/whyrusleeping/go-logging"
)

type Tx struct {
	from		string
	to		string
	amount		int
	signature 	string
}

type Block struct {
	Index     	int
	Timestamp 	string
	Signature   string
	Hash      	string
	PrevHash  	string
	Difficulty	int
	Nonce 		string
	Txs			[]Tx
}

const difficulty = 1
const broadcastInterval = 5 * time.Second
const miningInterval = 5 * time.Second

var Blockchain []Block
var unverifiedTxs []Tx
var mutex =&sync.Mutex{}

func calculateHash(block Block) string {
	record := strconv.Itoa(block.Index) + block.Timestamp + block.Signature + block.PrevHash + block.Nonce //add txs to be hashed (merkle root?)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func isHashValid(hash string, difficulty int) bool {
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hash, prefix)
}

func mineNewBlock() {
	for {
		time.Sleep(miningInterval)
		var txs []Tx
		//gets first available unverifiedTxs and verify them
		newBlock := generateBlock(Blockchain[len(Blockchain)-1], os.Getenv("SIG"), txs)
		if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
			mutex.Lock()
			Blockchain = append(Blockchain, newBlock)
			mutex.Unlock()
		}
	}
}

func generateBlock(oldBlock Block, signature string, txs []Tx) Block {
	var newBlock Block
	var hash string
	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Signature = signature
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Difficulty = difficulty
	newBlock.Txs = txs
	//add code to add txs and check their validity
	for i := 0; ; i++ {
		hex := fmt.Sprintf("%x", i)
		newBlock.Nonce = hex
		hash = calculateHash(newBlock)
		if isHashValid(hash, newBlock.Difficulty) {
		    newBlock.Hash = hash
		    spew.Dump(newBlock)
		    break
		} else {
			//fmt.Println(calculateHash(newBlock))
			time.Sleep(25 * time.Millisecond)
			continue		        
		}
	}

	return newBlock
}

func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index + 1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}
	hash := calculateHash(newBlock)
	if hash != newBlock.Hash {
		return false
	}
	//check difficulty
	if !isHashValid(hash, newBlock.Difficulty) {
		return false
	}
	//check validity of txs
	return true
}

//P2P stuff
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
	log.Printf("Full address: %s", fullAddr)
	log.Printf("Run \"go run p2p.go -l %d -d %s\" on a different terminal\n", listenPort+1, fullAddr)
	
	return basicHost, nil
}

func handleStream(s net.Stream) {
	log.Println("New stream connected")
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go broadcastBlockchain(rw)
	go broadcastUnverifiedTxs(rw)
	go receiveBlockchain(rw)
	go receiveUnverifiedTxs(rw)
	
}

func receiveBlockchain(rw *bufio.ReadWriter) {

	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {

			chain := make([]Block, 0)
			if err := json.Unmarshal([]byte(str), &chain); err != nil {
				log.Fatal(err)
			}

			mutex.Lock()
			if len(chain) > len(Blockchain) {
				Blockchain = chain
			}
			mutex.Unlock()
		}
			
	}
}

func receiveUnverifiedTxs(rw *bufio.ReadWriter) {

	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {

			uTxs := make([]Tx, 0)
			if err := json.Unmarshal([]byte(str), &uTxs); err != nil {
				log.Fatal(err)
			}

			mutex.Lock()

			//this is the complicated part. Do we take unions of both mine and received uTxs?
			//right now we replace if there are more, obv wrong!
			if len(uTxs) > len(unverifiedTxs) {
				unverifiedTxs = uTxs
				
			}
			mutex.Unlock()
		}
	}
}

func broadcastBlockchain(rw *bufio.ReadWriter) {
	//broadcast your blockchain version every broadcastInterval seconds.
	//We need to broadcast also unverified txs
	for {
		time.Sleep(broadcastInterval)
		
		mutex.Lock()
		bytes, err := json.Marshal(Blockchain)
		if err != nil {
			log.Println(err)
		}
		mutex.Unlock()

		mutex.Lock()
		rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
		rw.Flush()
		mutex.Unlock()

	}
}

func broadcastUnverifiedTxs(rw *bufio.ReadWriter) {
	//broadcast unverified txs
	for {
		time.Sleep(broadcastInterval)
		
		mutex.Lock()
		bytes, err := json.Marshal(unverifiedTxs)
		if err != nil {
			log.Println(err)
		}
		mutex.Unlock()

		mutex.Lock()
		rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
		rw.Flush()
		mutex.Unlock()
	}
}


//this function reads command from command line, as print Blockchain, unverified transactions and send transaction
func readCommand() {
	stdReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		spew.Dump(sendData)
		// var tx Tx

	 //    err = json.NewDecoder(os.Stdin).Decode(&tx)
	 //    if err != nil {
	 //        log.Fatal(err)
	 //    }
		// log.Println(tx)
		// bytes, err := json.Marshal(tx)
		// if err != nil {
		// 	log.Println(err)
		// }
		// log.Println(bytes)
		

		// spew.Dump(tx)

		// //print unverified txs
		// bytes, err := json.MarshalIndent(unverifiedTxs, "", "  ")
		// if err != nil {

		// 	log.Fatal(err)
		// }
		// // Green console color: 	\x1b[32m
		// // Reset console color: 	\x1b[0m
		// fmt.Printf("\x1b[32m%s\x1b[0m> ", string(bytes))
	}

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	t := time.Now()
	var txs []Tx
	genesisBlock := Block{}
	genesisBlock = Block{0, t.String(), os.Getenv("SIG"), calculateHash(genesisBlock), "", difficulty, "", txs}

	Blockchain = append(Blockchain, genesisBlock)

	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line
	defaultListen, err := strconv.Atoi(os.Getenv("ADDR"))
	if err != nil {
		log.Println(err, ". Running on default port 10000.")
		defaultListen = 10000
	}
	listenF := flag.Int("l", defaultListen, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	miner := flag.Bool("m", false, "set node as miner")
	flag.Parse()

	// if *listenF == 0 {
	// 	log.Fatal("Please provide a port to bind on with -l")
	// }

	// Make a host that listens on the given multiaddress
	ha, err := makeBasicHost(*listenF, *seed)
	if err != nil {
		log.Fatal(err)
	}
	if *miner {
		go mineNewBlock() 
	}
	go readCommand()
	if *target == "" {
		log.Println("listening for connections")
		// Set a stream handler on host A. /p2p/1.0.0 is
		// a user-defined protocol name.
		ha.SetStreamHandler("/p2p/1.0.0", handleStream)

		select {} // hang forever
		/**** This is where the listener code ends ****/
	} else {
		ha.SetStreamHandler("/p2p/1.0.0", handleStream)

		// The following code extracts target's peer ID from the
		// given multiaddress
		ipfsaddr, err := ma.NewMultiaddr(*target)
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
		ha.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

		log.Println("opening stream")
		// make a new stream from host B to host A
		// it should be handled on host A by the handler we set above because
		// we use the same /p2p/1.0.0 protocol
		s, err := ha.NewStream(context.Background(), peerid, "/p2p/1.0.0")
		if err != nil {
			log.Fatalln(err)
		}
		// Create a buffered stream so that read and writes are non blocking.
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		go broadcastBlockchain(rw)
		go broadcastUnverifiedTxs(rw)
		go receiveBlockchain(rw)
		go receiveUnverifiedTxs(rw)	
		
		select {} // hang forever

	}
}