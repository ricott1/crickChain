package main

import (
	"fmt"
	"time"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	)

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