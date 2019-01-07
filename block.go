package main

import (
	"fmt"
	"time"
	"crypto/sha256"
	"strconv"
	"math/rand"
	"math/bits"
	"math"
	"encoding/binary"
	)

type Block struct {
	Index     		int
	Timestamp 		int64
	POW_difficulty 	int
	POWS_difficulty int
	Signature   	string
	Hash      		[]byte
	HashInt32		uint32
	PrevHash  		[]byte
	Nonce 			string
	Txos			map[string]*UTXO
	Problem			int
	Solution		int
	HasSolution 	bool
}

func getBlockHash(block Block) []byte {
	record := strconv.Itoa(block.Index) + strconv.FormatInt(block.Timestamp, 10) + block.Signature + string(block.PrevHash) + block.Nonce //add utxos to be hashed (merkle root?)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hashed[:] //hex.EncodeToString(hashed)
}

func isHashValid(hash []byte, difficulty int) bool {
	source_uint := binary.BigEndian.Uint32(hash)
	return bits.LeadingZeros32(source_uint) >= difficulty
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

func getDifficulties(chain []Block, index int) (int, int) {
	pow_difficulty := float64(POW_DIFFICULTY)
	pows_difficulty := float64(POWS_DIFFICULTY)
	//start := chain[0].Timestamp
	//var stop int64
	//var duration int64
	var change float64
	var ratio float64
	var eta float64
	var timestampSolution int64
	var timestampClassic int64
	var base_pow_difficulty float64
	var base_pows_difficulty float64
	for i := 1; i < index; i++ {
	    if int64(i)%BLOCKS_PER_DIFFICULTY_UPDATE == 0 {
	       	ratio = getClassicBlocksRatio(chain, index)

	       	timestampSolution = getSolutionBlocksDuration(chain, index)
	       	timestampClassic = getClassicBlocksDuration(chain, index)
			eta = float64(timestampSolution)/float64(timestampClassic)
			change = (ratio + (1.0-ratio)*eta)/(ratio + (1.0-ratio)*ETA)

			base_pow_difficulty = pow_difficulty
			base_pows_difficulty = pows_difficulty
			pow_difficulty = pow_difficulty * change
			pows_difficulty = pows_difficulty + ETA * pow_difficulty - eta * (pow_difficulty - change)
			
			//enforce difficulty update limits
			// pow_difficulty = math.Round(math.Max(MAX_DIFFICULTY_CHANGE * base_pow_difficulty, math.Min(MIN_DIFFICULTY_CHANGE * base_pow_difficulty, pow_difficulty)))
			// pows_difficulty = math.Round(math.Max(MAX_DIFFICULTY_CHANGE * base_pows_difficulty, math.Min(MIN_DIFFICULTY_CHANGE * base_pows_difficulty, pows_difficulty)))
			pow_difficulty = math.Round(pow_difficulty)
			if pow_difficulty > base_pow_difficulty + MAX_DIFFICULTY_CHANGE {
				pow_difficulty = base_pow_difficulty + MAX_DIFFICULTY_CHANGE
			} else if pow_difficulty < base_pow_difficulty - MAX_DIFFICULTY_CHANGE {
				pow_difficulty = math.Max(base_pow_difficulty - MAX_DIFFICULTY_CHANGE, 1)
			}
			if pows_difficulty > base_pows_difficulty + MAX_DIFFICULTY_CHANGE {
				pows_difficulty = base_pows_difficulty + MAX_DIFFICULTY_CHANGE
			} else if pows_difficulty < base_pows_difficulty - MAX_DIFFICULTY_CHANGE {
				pows_difficulty = math.Max(base_pows_difficulty - MAX_DIFFICULTY_CHANGE, 1)
			}	    }
	}
	return int(pow_difficulty), int(pows_difficulty)
}

func getClassicBlocksRatio(chain []Block, index int) float64 {
	var r float64 = 0
	for i := 1; i < index; i++ {
		if !chain[i].HasSolution {
			r ++
		}	    
	}
	return r/(float64(index+1))
}

func getSolutionBlocksDuration(chain []Block, index int) int64 {
	var t int64 = 0
	for i := 1; i < index; i++ {
		if chain[i].HasSolution {
			t += chain[i].Timestamp - chain[i-1].Timestamp
		}	    
	}
	return t
}

func getClassicBlocksDuration(chain []Block, index int) int64 {
	var t int64 = 0
	for i := 1; i < index; i++ {
		if !chain[i].HasSolution {
			t += chain[i].Timestamp - chain[i-1].Timestamp
		}	    
	}
	return t
}

func generateBlock(oldBlock Block, signature string, txos map[string]*UTXO) Block {
	var newBlock Block
	var hash []byte
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
	var diff int
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
		
		if isHashValid(hash, diff) {
		    newBlock.Hash = hash
		    newBlock.HashInt32 = binary.BigEndian.Uint32(hash)
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

	if string(oldBlock.Hash) != string(newBlock.PrevHash) {
		return false
	}
	hash := getBlockHash(newBlock)
	if string(hash) != string(newBlock.Hash) {
		return false
	}
	// //check difficulty
	// pows_difficulty := getPOWSDifficulty(chain, newBlock.Index)
	// pow_difficulty := getPOWDifficulty(chain, newBlock.Index)
	pow_difficulty, pows_difficulty := getDifficulties(chain, newBlock.Index)

	if newBlock.POWS_difficulty != pows_difficulty {
		fmt.Println("Failed pows difficulty check", newBlock.POWS_difficulty, pows_difficulty)
		return false
	}
	if newBlock.POW_difficulty != pow_difficulty {
		fmt.Println("Failed pow difficulty check", newBlock.POW_difficulty, pow_difficulty)
		return false
	}
	bestSolution := findBestSolution(chain, newBlock.Index)
	if hasValidSolution(newBlock, bestSolution) {
		if !isHashValid(hash, pows_difficulty) {
			fmt.Println("Failed pows valid check", hash, pows_difficulty)
			return false
		} 
	} else {
		if !isHashValid(hash, pow_difficulty) {
			fmt.Println("Failed pow valid check", hash, pow_difficulty)
			return false
		}
	}	
	//check validity of utxos
	return true
}
