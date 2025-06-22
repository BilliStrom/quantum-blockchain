package main

import (
	"fmt"
	"log"
	"quantum-blockchain/blockchain"
)

func main() {
	fmt.Println("🟣 Quantum Blockchain v1.0")
	fmt.Println("Creator: BilliStrom | 22.06.2025")

	bc := blockchain.InitBlockchain()
	defer bc.Database.Close()

	// Создание кошельков
	walletA := blockchain.NewWallet()
	walletB := blockchain.NewWallet()

	// Добавление стейка
	bc.AddStake(walletA.Address(), big.NewInt(1000))
	bc.AddStake(walletB.Address(), big.NewInt(500))

	// Генерация транзакций
	tx := blockchain.NewTransaction(walletA, walletB.Address(), 50)
	bc.AddBlock([]*blockchain.Transaction{tx})

	log.Println("✅ Blockchain запущен!")
}