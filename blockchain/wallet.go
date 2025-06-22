package blockchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewWallet() *Wallet {
	private, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	public := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return &Wallet{*private, public}
}

func (w Wallet) Address() []byte {
	pubHash := sha256.Sum256(w.PublicKey)
	hasher := ripemd160.New()
	hasher.Write(pubHash[:])
	publicRipMD := hasher.Sum(nil)
	return append([]byte{version}, publicRipMD...)
}