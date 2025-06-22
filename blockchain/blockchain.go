package blockchain

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"
	
	"github.com/dgraph-io/badger"
)

type Block struct {
	Timestamp    int64
	Transactions []*Transaction
	PrevHash     []byte
	Hash         []byte
	Nonce        int
	Validator    []byte
}

type Blockchain struct {
	LastHash    []byte
	Database    *badger.DB
	Validators  []Validator
	mu          sync.Mutex
}

type Validator struct {
	Address []byte
	Stake   *big.Int
}

func InitBlockchain(dataDir string) (*Blockchain, error) {
	var lastHash []byte
	
	opts := badger.DefaultOptions(dataDir)
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	err = db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte("lh"))
		if err == badger.ErrKeyNotFound {
			genesis := GenesisBlock()
			log.Println("✅ Genesis block created")
			
			err = txn.Set(genesis.Hash, genesis.Serialize())
			if err != nil {
				return err
			}
			
			err = txn.Set([]byte("lh"), genesis.Hash)
			lastHash = genesis.Hash
			return err
		}
		
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
	})

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize blockchain: %v", err)
	}

	bc := &Blockchain{
		LastHash:   lastHash,
		Database:   db,
		Validators: make([]Validator, 0),
	}
	
	bc.InitValidators()
	
	return bc, nil
}

func GenesisBlock() *Block {
	coinbase := CoinbaseTx([]byte("GENESIS_ADDRESS"), "Genesis block")
	return &Block{
		Timestamp:    time.Now().Unix(),
		Transactions: []*Transaction{coinbase},
		PrevHash:     []byte{},
		Hash:         []byte("GENESIS_HASH"),
		Validator:    []byte("GENESIS_VALIDATOR"),
	}
}

func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)
	if err := encoder.Encode(b); err != nil {
		log.Panic("serialization error:", err)
	}
	return res.Bytes()
}

func DeserializeBlock(data []byte) (*Block, error) {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&block); err != nil {
		return nil, err
	}
	return &block, nil
}

func (bc *Blockchain) InitValidators() {
	bc.AddStake([]byte("VALIDATOR_ADDR_1"), big.NewInt(1000))
	bc.AddStake([]byte("VALIDATOR_ADDR_2"), big.NewInt(500))
}

func (bc *Blockchain) AddStake(address []byte, amount *big.Int) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	for i, v := range bc.Validators {
		if bytes.Equal(v.Address, address) {
			bc.Validators[i].Stake = new(big.Int).Add(v.Stake, amount)
			return
		}
	}
	
	bc.Validators = append(bc.Validators, Validator{
		Address: address,
		Stake:   amount,
	})
}

func (bc *Blockchain) SelectValidator() ([]byte, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	if len(bc.Validators) == 0 {
		return nil, errors.New("no validators available")
	}
	
	totalStake := new(big.Int)
	for _, v := range bc.Validators {
		totalStake.Add(totalStake, v.Stake)
	}
	
	randValue, err := rand.Int(rand.Reader, totalStake)
	if err != nil {
		return nil, err
	}
	
	current := new(big.Int)
	for _, v := range bc.Validators {
		current.Add(current, v.Stake)
		if current.Cmp(randValue) > 0 {
			return v.Address, nil
		}
	}
	
	return bc.Validators[len(bc.Validators)-1].Address, nil
}

func (bc *Blockchain) AddBlock(transactions []*Transaction) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	var lastHash []byte
	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
	})
	
	if err != nil {
		return err
	}
	
	validator, err := bc.SelectValidator()
	if err != nil {
		return err
	}
	
	log.Printf("🔒 Validator selected: %x (Stake: %d)", validator, bc.GetStake(validator))
	
	newBlock := &Block{
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		PrevHash:     lastHash,
		Validator:    validator,
	}
	
	newBlock.Hash = newBlock.CalculateHash()
	
	err = bc.Database.Update(func(txn *badger.Txn) error {
		if err := txn.Set(newBlock.Hash, newBlock.Serialize()); err != nil {
			return err
		}
		if err := txn.Set([]byte("lh"), newBlock.Hash); err != nil {
			return err
		}
		bc.LastHash = newBlock.Hash
		return nil
	})
	
	if err != nil {
		return err
	}
	
	log.Printf("🔗 Block %x added", newBlock.Hash)
	return nil
}

func (b *Block) CalculateHash() []byte {
	data := bytes.Join(
		[][]byte{
			b.PrevHash,
			[]byte(time.Unix(b.Timestamp, 0).String()),
			b.HashTransactions(),
			big.NewInt(int64(b.Nonce)).Bytes(),
			b.Validator,
		},
		[]byte{},
	)
	
	hash := sha256.Sum256(data)
	return hash[:]
}

func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	
	// В реальной реализации используйте Merkle tree
	combined := bytes.Join(txHashes, []byte{})
	hash := sha256.Sum256(combined)
	return hash[:]
}

func (bc *Blockchain) GetStake(address []byte) *big.Int {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	for _, v := range bc.Validators {
		if bytes.Equal(v.Address, address) {
			return v.Stake
		}
	}
	return big.NewInt(0)
}

func (bc *Blockchain) GetLastBlock() (*Block, error) {
	var blockBytes []byte
	
	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(bc.LastHash)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			blockBytes = append([]byte{}, val...)
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return DeserializeBlock(blockBytes)
}

func (bc *Blockchain) GetBalance(address []byte) *big.Int {
	balance := big.NewInt(0)
	
	// Упрощенная реализация - в реальном проекте нужно учитывать UTXO
	err := bc.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				block, err := DeserializeBlock(val)
				if err != nil {
					return err
				}
				
				for _, tx := range block.Transactions {
					if bytes.Equal(tx.To, address) {
						balance.Add(balance, tx.Value)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	
	if err != nil {
		log.Printf("Error calculating balance: %v", err)
	}
	
	return balance
}

func (bc *Blockchain) Close() {
	if err := bc.Database.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
}