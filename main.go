package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"sync"
	//"reflect"

	"github.com/joho/godotenv"
	//"github.com/davecgh/go-spew/spew"
	golog "github.com/ipfs/go-log"
	gologging "github.com/whyrusleeping/go-logging"
)

type BroadcastData struct {
	Blockchain 	[]Block
	UTXOs 		map[string]*UTXO
}

const POW_DIFFICULTY = 1
const BROADCAST_INTERVAL = 5 * time.Second
const MINING_INTERVAL = 5 * time.Second
const UTXO_PER_BLOCK = 5

var Blockchain []Block
//this is wrong. The UTXOs should represent basically the money in the system, not the proposed txs. Or better, when someone submits a tansaction, if valid (i.e. if money present in UTXOs) it just update the UTXOs
var UTXOs map[string]*UTXO //notice the *, meaning that this is a map of pointers. To assign we have to prepend & to the value. The map ensures that we don't have duplicates
var Data BroadcastData
var Wallets map[string]*Wallet
var Mutex =&sync.Mutex{}

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
	seed := flag.Int64("s", 0, "set random seed for id generation")
	miner := flag.Bool("m", false, "set node as miner")
	flag.Parse()

	fmt.Println(*seed)
	// Make a host that listens on the given multiaddress
	host, err := makeBasicHost(*listenF, *seed)
	if err != nil {
		log.Fatal(err)
	}
	if *miner {
		go mineNewBlock() 
	}
	go readCommand()
	if *target == "" {
		//Listen for new connections
		createListener(host)
		select {} // hang forever
	} else {
		connectToPeer(host, *target)
		select {} // hang forever

	}
}