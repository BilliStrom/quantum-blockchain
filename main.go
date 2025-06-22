package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"quantum-blockchain/blockchain"
	"quantum-blockchain/rpc"
	"syscall"
)

func main() {
	dataDir := flag.String("datadir", "./qdata", "Data directory for blockchain")
	rpcPort := flag.String("rpcport", "8545", "RPC server port")
	validator := flag.Bool("validator", false, "Run as validator node")
	flag.Parse()

	fmt.Println("🟣 Quantum Blockchain v1.0")
	fmt.Println("Creator: BilliStrom | 22.06.2025")

	bc, err := blockchain.InitBlockchain(*dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize blockchain: %v", err)
	}
	defer bc.Close()

	// Пример добавления транзакций
	if *validator {
		log.Println("🔐 Running in validator mode")
		wallet := blockchain.NewWallet()
		bc.AddStake(wallet.Address(), big.NewInt(1000))
	}

	// Запуск RPC сервера
	go rpc.Start(*rpcPort, bc)

	// Обработка сигналов для корректного завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("🛑 Shutting down...")
}