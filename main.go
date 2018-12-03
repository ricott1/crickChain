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
	//"github.com/davecgh/go-spew/spew"
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

type UTXO struct {
	Timestamp 	string
	From		string
	To			string
	Amount		int
	Id 			string
	PublicKey 	string
	Fee			int
}

type Block struct {
	Index     	int
	Timestamp 	string
	Signature   string
	Hash      	string
	PrevHash  	string
	Difficulty	int
	Nonce 		string
	Txos		map[string]*UTXO
}

type BroadcastData struct {
	Blockchain 	[]Block
	UTXOs 		map[string]*UTXO
}

const POW_DIFFICULTY = 1
const BROADCAST_INTERVAL = 5 * time.Second
const MINING_INTERVAL = 5 * time.Second
const UTXO_PER_BLOCK = 5

var Blockchain []Block
var UTXOs map[string]*UTXO //notice the *, meaning that this is a map of pointers. To assign we have to prepend & to the value. The map ensures that we don't have duplicates
var Data BroadcastData
var Mutex =&sync.Mutex{}

func getBlockHash(block Block) string {
	record := strconv.Itoa(block.Index) + block.Timestamp + block.Signature + block.PrevHash + block.Nonce //add utxos to be hashed (merkle root?)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func isPoWValid(hash string, difficulty int) bool {
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hash, prefix)
}

func isUTXOValid(utxo *UTXO) bool {
	//add to check if it was not included in previous block
	for i := len(Blockchain) - 1; i >= 0; i-- {
	    block := Blockchain[i]
	    for k, _ := range block.Txos {
	    	if utxo.Id == k {
	    		return false
	    	}
	    }
	}

	return true
}

func newUTXO(to string) UTXO {
	from := os.Getenv("SIG")
	t := time.Now().String()
	record := from + to + t
	h := sha256.New()
	h.Write([]byte(record))
	id := hex.EncodeToString(h.Sum(nil))
	return UTXO{t, from, to, 4, id, from, 5}
}

func filterUTXOs(utxos map[string]*UTXO) map[string]*UTXO{
	for k, v := range utxos {
			if k != v.Id {
				delete(utxos, k)
			} else if !isUTXOValid(v) {
				delete(utxos, k)
			}
		}
	return utxos
}

func mineNewBlock() {
	for {
		time.Sleep(MINING_INTERVAL)
		//take the UTXO, filter them to be sure, and collect the first UTXO_PER_BLOCK into the block. One could instead order them to include those with higeher fees first.
		txos := make(map[string]*UTXO)
		//log.Println(UTXOs)
		//log.Println(filterUTXOs(UTXOs))
		for k, v := range filterUTXOs(UTXOs) {
			    txos[k] = v
			    if len(txos) > UTXO_PER_BLOCK {
			    	break
			    }
			}
		newBlock := generateBlock(Blockchain[len(Blockchain)-1], os.Getenv("SIG"), txos)
		if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
			Mutex.Lock()
			Blockchain = append(Blockchain, newBlock)
			Mutex.Unlock()
		}
	}
}

func generateBlock(oldBlock Block, signature string, txos map[string]*UTXO) Block {
	var newBlock Block
	var hash string
	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Signature = signature
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Difficulty = POW_DIFFICULTY
	newBlock.Txos = txos
	//add code to add utxos and check their validity
	for i := 0; ; i++ {
		hex := fmt.Sprintf("%x", i)
		newBlock.Nonce = hex
		hash = getBlockHash(newBlock)
		if isPoWValid(hash, newBlock.Difficulty) {
		    newBlock.Hash = hash
		    //spew.Dump(newBlock)
		    break
		} else {
			//fmt.Println(getBlockHash(newBlock))
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
	hash := getBlockHash(newBlock)
	if hash != newBlock.Hash {
		return false
	}
	//check difficulty
	if !isPoWValid(hash, newBlock.Difficulty) {
		return false
	}
	//check validity of utxos
	return true
}

func printCommand(bytes []byte) {
	// Green console color: 	\x1b[32m
	// Reset console color: 	\x1b[0m
	fmt.Printf("\x1b[32m%s\x1b[0m ", string(bytes))
}

//this function reads command from command line, as print Blockchain, unverified transactions and send transaction
func readCommand() {
	stdReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		command := strings.TrimSpace(sendData)
		//spew.Dump(command)
		if command == "" {
			continue
		}
		if command == "bc" {
			bytes, err := json.MarshalIndent(Blockchain, "", "  ")
			if err != nil {
				log.Println(err)
			} else{
				printCommand(bytes)
			}
		} else if command == "ut" {
			bytes, err := json.MarshalIndent(UTXOs, "", "  ")
			if err != nil {
				log.Println(err)
			} else{
				printCommand(bytes)
			}
		} else if commands := strings.Fields(command); commands[0] == "nut" && len(commands) == 2 {
			to := commands[1]
			newUTXO := newUTXO(to)
			Mutex.Lock()
			UTXOs[newUTXO.Id] = &newUTXO
			Mutex.Unlock()
			bytes, err := json.MarshalIndent(newUTXO, "", "  ")
			if err != nil {
				log.Println(err)
			} else{
				printCommand(bytes)
			}
		} else if command == "h" {
			fmt.Printf(`
bc : Display Blockchain
ut : Display unverified transaction outputs (UTXOs)
nut : Create new UTXO
h  : Display this helper
q  : Quit`)
		} else if command == "q" {
			os.Exit(1)
		} else {
			fmt.Printf("Invalid command, call h for help")
		}
	}
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
	fmt.Printf("go run main.go -l %d -d %s\n", listenPort+1, fullAddr)
	
	return basicHost, nil
}

func handleStream(s net.Stream) {
	log.Println("New stream connected")
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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	t := time.Now()
	utxos := make(map[string]*UTXO)
	genesisBlock := Block{}
	genesisBlock = Block{0, t.String(), os.Getenv("SIG"), getBlockHash(genesisBlock), "", POW_DIFFICULTY, "", utxos}

	Blockchain = append(Blockchain, genesisBlock)
	UTXOs = make(map[string]*UTXO, 0)

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

		go broadcastData(rw)
		go receiveBroadcastData(rw)
		
		select {} // hang forever

	}
}