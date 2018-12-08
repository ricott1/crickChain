package main

import (
	"log"
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
)



type Wallet struct {
	Address 	string //last 20 bytes of hash of public key
	Balance		int		//it's here for convenience, but should be calculated from the UTXOs
	PublicKey	string 
	privateKey 	string
}

func newWallet(pwd string) Wallet {
	pri, err := PrivateKeyToEncryptedPEM(2028, pwd)
	if err != nil {
		log.Fatal(err)
	}
	pub := pri
	add := pub[len(pub)-21:len(pub)-1]
	return Wallet{string(add), 0, string(pub), string(pri)}
}

func PrivateKeyToEncryptedPEM(bits int, pwd string) ([]byte, error) {
    // Generate the key of length bits
    key, err := rsa.GenerateKey(rand.Reader, bits)
    if err != nil {
        return nil, err
    }

    // Convert it to pem
    block := &pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: x509.MarshalPKCS1PrivateKey(key),
    }

    // Encrypt the pem
    if pwd != "" {
        block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(pwd), x509.PEMCipherAES256)
        if err != nil {
            return nil, err
        }
    }

    return pem.EncodeToMemory(block), nil
}