package main

import (
	"fmt"
	"time"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"math/rand"
	"math"
	)

type Block struct {
	Index     		int
	Timestamp 		int64
	POW_difficulty 	int64
	POWS_difficulty int64
	Signature   	string
	Hash      		string
	PrevHash  		string
	Nonce 			string
	Txos			map[string]*UTXO
	Problem			int
	Solution		int
	HasSolution 	bool
}

func getBlockHash(block Block) string {
	record := strconv.Itoa(block.Index) + strconv.FormatInt(block.Timestamp, 10) + block.Signature + block.PrevHash + block.Nonce //add utxos to be hashed (merkle root?)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func isPoWValid(hash string, difficulty int64) bool {
	prefix := strings.Repeat("0", int(difficulty))
	return strings.HasPrefix(hash, prefix)
}

func hasValidSolution(newBlock Block, bestSolution int) bool {
	if newBlock.Solution < bestSolution {
		return true
	}
	return false
}

func findBestSolution(chain []Block, index int) int {
	bsol := 10
	var sol int
	var block Block
	for i := 0; i < index; i++ {
	    block = chain[i]
	    sol = block.Solution
	    if sol < bsol {
	    	bsol = sol
	    }
	}
	return bsol
}

func isBlockchainValid(chain []Block) bool {
	for i := len(chain) - 1; i > 0; i-- {
	    if !isBlockValid(chain, chain[i], chain[i-1]) {
	    	fmt.Println("Wrong", i)
	    	return false
	    }
	}
	return true
}

func getDifficulties(chain []Block, index int) (int64, int64) {
	pow_difficulty := float64(POW_DIFFICULTY)
	pows_difficulty := float64(POWS_DIFFICULTY)
	start := chain[0].Timestamp
	var stop int64
	var duration int64
	var change float64
	//var ratio float64
	//var eta float64
	//var timestampSolution int64
	//var timestampClassic int64
	for i := 1; i < index; i++ {
	    if int64(i)%BLOCKS_PER_DIFFICULTY_UPDATE == 0 {
	    	stop = chain[i].Timestamp
	    	duration = stop - start
	    	change = math.Max(MAX_DIFFICULTY_CHANGE, math.Min(MIN_DIFFICULTY_CHANGE, float64(BLOCKS_PER_DIFFICULTY_UPDATE) * NANOSECONDS_PER_MINUTE / BLOCKS_PER_MINUTE / float64(duration)))
	    	pow_difficulty = pow_difficulty * change
	    	pows_difficulty = pows_difficulty * change

//new stuff
    //    	ratio = getBlocksWithSolution(chain, index)
	// 		eta = float64(timestampSolution)/float64(timestampClassic)
	// 		change = (b + (1-b)*eta)/(b + (1-b)*ETA)
	// 		pow_difficulty = pow_difficulty + change
	//		pows_difficulty = pows_difficulty + ETA * pow_difficulty - eta * (pow_difficulty - change)
	    	start = stop
	    }
	}
	return int64(math.Round(pow_difficulty)), int64(math.Round(pows_difficulty))
}

// func getPOWDifficulty(chain []Block, index int) int64 {
// 	difficulty := POW_DIFFICULTY
// 	start := chain[0].Timestamp
// 	var stop int64
// 	var duration int64
// 	var ratio float64
// 	var change float64
// 	var eta float64
// 	for i := 1; i < index; i++ {
// 	    if int64(i)%BLOCKS_PER_DIFFICULTY_UPDATE == 0 {
// 	    	stop = chain[i].Timestamp
// 	    	duration = stop - start
// 	    	//change = math.Max(MAX_DIFFICULTY_CHANGE, math.Min(MIN_DIFFICULTY_CHANGE, float64(BLOCKS_PER_DIFFICULTY_UPDATE * NANOSECONDS_PER_MINUTE / BLOCKS_PER_MINUTE / duration)))
// 	    	//difficulty = int64(float64(difficulty) * change)
// 	    	ratio = getBlocksWithSolution(chain, index)
// 		eta = 
// 		change = (b + (1-b)*eta)/(b + (1-b)*ETA)
// 		difficulty = difficulty + change
// 		start = stop
// 	    }
// 	}
// 	return difficulty
// }

// func getPOWSDifficulty(chain []Block, index int) int64 {
// 	difficulty := POWS_DIFFICULTY
// 	start := chain[0].Timestamp
// 	var stop int64
// 	var duration int64

// 	var change float64
// 	// var T float64 := 
// 	for i := 1; i < index; i++ {
// 	    if int64(i)%BLOCKS_PER_DIFFICULTY_UPDATE == 0 {
// 	    	stop = chain[i].Timestamp
// 	    	duration = stop - start
// 	    	change = math.Max(MAX_DIFFICULTY_CHANGE, math.Min(MIN_DIFFICULTY_CHANGE, float64(BLOCKS_PER_DIFFICULTY_UPDATE * NANOSECONDS_PER_MINUTE / BLOCKS_PER_MINUTE / duration)))
// 	    	difficulty = int64(float64(difficulty) * change)
// 	    	start = stop
// 	    }
// 	}
// 	return difficulty
// }

func getBlocksWithSolutionRatio(chain []Block, index int) float64 {
	var r float64 = 0
	for i := 1; i < index; i++ {
		if chain[i].HasSolution {
			r ++
		}	    
	}
	return r/(float64(index+1))
}

func getBlocksWithSolutionDuration(chain []Block, index int) int64 {
	
}

func generateBlock(oldBlock Block, signature string, txos map[string]*UTXO) Block {
	var newBlock Block
	var hash string
	newBlock.Index = oldBlock.Index + 1
	
	// newBlock.POW_difficulty = getPOWDifficulty(Blockchain, newBlock.Index)
	// newBlock.POWS_difficulty = getPOWSDifficulty(Blockchain, newBlock.Index)
	newBlock.POW_difficulty, newBlock.POWS_difficulty = getDifficulties(Blockchain, newBlock.Index)
	newBlock.Signature = signature
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Problem = 10
	bestSolution := findBestSolution(Blockchain, newBlock.Index)
	t := time.Now().UnixNano()
	newBlock.Timestamp = t
	
	
	newBlock.Txos = txos
	//add code to add utxos and check their validity
	var diff int64
	for i := 0; ; i++ {
		newBlock.Solution = bestSolution + rand.Intn(4) - 2
		hex := fmt.Sprintf("%x", i)
		newBlock.Nonce = hex
		hash = getBlockHash(newBlock)
		if hasValidSolution(newBlock, bestSolution) { 
			diff = newBlock.POWS_difficulty
			newBlock.HasSolution = true
		} else {
			diff = newBlock.POW_difficulty
			newBlock.HasSolution = false
		}
		
		if isPoWValid(hash, diff) {
		    newBlock.Hash = hash
		    //spew.Dump(newBlock)
		    break
		} else {
			//fmt.Println(getBlockHash(newBlock))
			time.Sleep(MINING_INTERVAL)
			continue		        
		}
	}

	return newBlock
}

func isBlockValid(chain []Block, newBlock Block, oldBlock Block) bool {
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
	// //check difficulty
	// pows_difficulty := getPOWSDifficulty(chain, newBlock.Index)
	// pow_difficulty := getPOWDifficulty(chain, newBlock.Index)
	pow_difficulty, pows_difficulty := getDifficulties(Blockchain, len(Blockchain))

	if newBlock.POWS_difficulty != pows_difficulty {
		fmt.Println("Failed pows check", newBlock.POWS_difficulty, pows_difficulty)
		return false
	}
	if newBlock.POW_difficulty != pow_difficulty {
		fmt.Println("Failed pow check", newBlock.POW_difficulty, pow_difficulty)
		return false
	}
	bestSolution := findBestSolution(chain, newBlock.Index)
	if hasValidSolution(newBlock, bestSolution) {
		if !isPoWValid(hash, pows_difficulty) {
			fmt.Println("Failed pows valid check", hash, pows_difficulty)
			return false
		} 
	} else {
		if !isPoWValid(hash, pow_difficulty) {
			fmt.Println("Failed pow valid check", hash, pow_difficulty)
			return false
		}
	}	
	//check validity of utxos
	return true
}
