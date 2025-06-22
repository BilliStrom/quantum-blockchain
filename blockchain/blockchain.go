package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
	"sync"
	"time"
	
	"github.com/dgraph-io/badger"
)

// Block представляет структуру блока в блокчейне
type Block struct {
	Timestamp    int64
	Transactions []*Transaction
	PrevHash     []byte
	Hash         []byte
	Nonce        int
	Validator    []byte
}

// Blockchain структура основной цепочки
type Blockchain struct {
	LastHash    []byte
	Database    *badger.DB
	Validators  []Validator
	mu          sync.Mutex
}

// Validator представляет участника стейкинга
type Validator struct {
	Address []byte
	Stake   *big.Int
}

// Инициализация блокчейна
func InitBlockchain(dataDir string) *Blockchain {
	var lastHash []byte
	
	// Настройки BadgerDB
	opts := badger.DefaultOptions(dataDir)
	opts.Logger = nil // Отключаем логгер для чистоты вывода

	db, err := badger.Open(opts)
	if err != nil {
		log.Panic(err)
	}

	// Проверяем существование блокчейна
	err = db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte("lh"))
		if err == badger.ErrKeyNotFound {
			// Создаем генезис-блок
			genesis := GenesisBlock()
			log.Println("✅ Genesis block created")
			
			err = txn.Set(genesis.Hash, genesis.Serialize())
			if err != nil {
				return err
			}
			
			err = txn.Set([]byte("lh"), genesis.Hash)
			lastHash = genesis.Hash
			return err
		} else {
			// Загружаем последний хеш
			item, err := txn.Get([]byte("lh"))
			if err != nil {
				return err
			}
			
			err = item.Value(func(val []byte) error {
				lastHash = append([]byte{}, val...)
				return nil
			})
			return err
		}
	})

	if err != nil {
		log.Panic(err)
	}

	bc := &Blockchain{
		LastHash:   lastHash,
		Database:   db,
		Validators: make([]Validator, 0),
	}
	
	// Инициализируем валидаторов
	bc.InitValidators()
	
	return bc
}

// Создание генезис-блока
func GenesisBlock() *Block {
	return &Block{
		Timestamp:    time.Now().Unix(),
		Transactions: []*Transaction{CoinbaseTx([]byte("GENESIS_ADDRESS"), "Genesis block")},
		PrevHash:     []byte{},
		Hash:         []byte("GENESIS_HASH"),
		Validator:    []byte("GENESIS_VALIDATOR"),
	}
}

// Сериализация блока
func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)
	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}
	return res.Bytes()
}

// Десериализация блока
func DeserializeBlock(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}
	return &block
}

// Инициализация валидаторов
func (bc *Blockchain) InitValidators() {
	// Пример начальных валидаторов
	bc.AddStake([]byte("VALIDATOR_ADDR_1"), big.NewInt(1000))
	bc.AddStake([]byte("VALIDATOR_ADDR_2"), big.NewInt(500))
}

// Добавление стейка
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

// Выбор валидатора (упрощенный алгоритм)
func (bc *Blockchain) SelectValidator() []byte {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	if len(bc.Validators) == 0 {
		return []byte("DEFAULT_VALIDATOR")
	}
	
	// Простой выбор валидатора с наибольшим стейком
	selected := bc.Validators[0].Address
	maxStake := bc.Validators[0].Stake
	
	for _, v := range bc.Validators[1:] {
		if v.Stake.Cmp(maxStake) > 0 {
			maxStake = v.Stake
			selected = v.Address
		}
	}
	
	return selected
}

// Добавление нового блока
func (bc *Blockchain) AddBlock(transactions []*Transaction) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	var lastHash []byte
	
	// Получаем последний хеш
	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}
		
		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
		return err
	})
	
	if err != nil {
		log.Panic(err)
	}
	
	// Выбираем валидатора
	validator := bc.SelectValidator()
	log.Printf("🔒 Validator selected: %x (Stake: %d)\n", validator, bc.GetStake(validator))
	
	// Создаем новый блок
	newBlock := &Block{
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		PrevHash:     lastHash,
		Validator:    validator,
	}
	
	// Рассчитываем хеш блока
	newBlock.Hash = newBlock.CalculateHash()
	
	// Сохраняем в базу данных
	err = bc.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			return err
		}
		
		err = txn.Set([]byte("lh"), newBlock.Hash)
		bc.LastHash = newBlock.Hash
		return err
	})
	
	if err != nil {
		log.Panic(err)
	}
	
	log.Printf("🔗 Block %x added\n", newBlock.Hash)
}

// Расчет хеша блока
func (b *Block) CalculateHash() []byte {
	header := bytes.Join(
		[][]byte{
			b.PrevHash,
			[]byte(time.Unix(b.Timestamp, 0).String()),
			b.HashTransactions(),
			big.NewInt(int64(b.Nonce)).Bytes(),
			b.Validator,
		},
		[]byte{},
	)
	
	hash := sha256.Sum256(header)
	return hash[:]
}

// Хеширование транзакций
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	
	tree := NewMerkleTree(txHashes)
	return tree.RootNode.Data
}

// Получение стейка валидатора
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

// Получение последнего блока
func (bc *Blockchain) GetLastBlock() *Block {
	var lastBlock *Block
	
	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(bc.LastHash)
		if err != nil {
			return err
		}
		
		err = item.Value(func(val []byte) error {
			lastBlock = DeserializeBlock(val)
			return nil
		})
		return err
	})
	
	if err != nil {
		log.Panic(err)
	}
	
	return lastBlock
}

// Закрытие базы данных
func (bc *Blockchain) Close() {
	if err := bc.Database.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
}