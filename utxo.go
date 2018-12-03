package main

import (
	"time"
	"os"
	"crypto/sha256"
	"encoding/hex"
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