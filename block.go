package main

import (
	"fmt"
	"time"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"math/rand"
	)

type Block struct {
	Index     	int
	Timestamp 	string
	Signature   string
	Hash      	string
	PrevHash  	string
	Nonce 		string
	Txos		map[string]*UTXO
	Problem		int
	Solution	int
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

func hasValidSolution(newBlock Block, bestSolution int) bool {
	if newBlock.Solution < bestSolution {
		return true
	}
	return false
}

func findBestSolution() int {
	bsol := 10
	var sol int
	var block Block
	for i := len(Blockchain) - 1; i >= 0; i-- {
	    block = Blockchain[i]
	    sol = block.Solution
	    if sol < bsol {
	    	bsol = sol
	    }
	}
	return bsol
}

func generateBlock(oldBlock Block, signature string, txos map[string]*UTXO) Block {
	var newBlock Block
	var hash string
	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Signature = signature
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Problem = 10
	bestSolution := findBestSolution()
	
	newBlock.Solution = bestSolution + rand.Intn(4) - 2
	newBlock.Txos = txos
	//add code to add utxos and check their validity
	if hasValidSolution(newBlock, bestSolution) {
		for i := 0; ; i++ {
			hex := fmt.Sprintf("%x", i)
			newBlock.Nonce = hex
			hash = getBlockHash(newBlock)
			if isPoWValid(hash, POWS_DIFFICULTY) {
			    newBlock.Hash = hash
			    //spew.Dump(newBlock)
			    break
			} else {
				//fmt.Println(getBlockHash(newBlock))
				time.Sleep(MINING_INTERVAL)
				continue		        
			}
		}
	} else {
		for i := 0; ; i++ {
			hex := fmt.Sprintf("%x", i)
			newBlock.Nonce = hex
			hash = getBlockHash(newBlock)
			if isPoWValid(hash, POW_DIFFICULTY) {
			    newBlock.Hash = hash
			    //spew.Dump(newBlock)
			    break
			} else {
				//fmt.Println(getBlockHash(newBlock))
				time.Sleep(25 * time.Millisecond)
				continue		        
			}
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
	bestSolution := findBestSolution()
	if hasValidSolution(newBlock, bestSolution) {
		if !isPoWValid(hash, POWS_DIFFICULTY) {
			return false
		} 
	} else if !isPoWValid(hash, POW_DIFFICULTY) {
				return false
	}	
	//check validity of utxos
	return true
}
