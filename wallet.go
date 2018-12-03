package main

type Wallet struct {
	address 	string //last 20 bytes of hash of public key
	balance		int
	publicKey	string 
	privateKey 	string
}