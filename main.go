package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"quantum-blockchain/blockchain"
	"quantum-blockchain/p2p"
	"quantum-blockchain/rpc"
	"syscall"
	"time"
)

func main() {
	// Параметры командной строки
	rpcPort := flag.Int("rpc", 8545, "RPC server port")
	p2pPort := flag.Int("p2p", 4001, "P2P network port")
	bootnodes := flag.String("bootnodes", "", "Comma-separated list of bootnodes")
	dataDir := flag.String("datadir", "./qdata", "Blockchain data directory")
	flag.Parse()

	fmt.Println("🚀 Quantum Blockchain v1.1")
	fmt.Println("🛰️ P2P Network Enabled")

	// Инициализация блокчейна
	bc, err := blockchain.InitBlockchain(*dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize blockchain: %v", err)
	}
	defer bc.Close()

	// Создание P2P сети
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	net, err := p2p.NewNetwork(ctx, *p2pPort)
	if err != nil {
		log.Fatalf("Failed to create P2P network: %v", err)
	}

	// Подключение к bootnodes
	if *bootnodes != "" {
		for _, addr := range strings.Split(*bootnodes, ",") {
			if addr == "" {
				continue
			}
			if err := net.Connect(ctx, addr); err != nil {
				log.Printf("Failed to connect to bootnode %s: %v", addr, err)
			}
			time.Sleep(1 * time.Second)
		}
	}

	// Запуск обработчика сообщений
	go bc.StartSyncHandler(net)

	// Запуск RPC сервера
	go func() {
		if err := rpc.Start(fmt.Sprint(*rpcPort), bc); err != nil {
			log.Fatalf("RPC server failed: %v", err)
		}
	}()

	// Бродкаст нового блока при создании
	bc.SetBroadcastFunc(func(block *blockchain.Block) {
		blockMsg := p2p.BlockMessage{Block: block}
		net.Broadcast(p2p.MsgBlock, p2p.EncodeGob(blockMsg))
	})

	// Обработка сигналов завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	
	log.Println("🛑 Shutting down node...")
	cancel()
	time.Sleep(1 * time.Second) // Даем время на завершение операций
}