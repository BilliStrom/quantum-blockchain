package rpc

import (
	"encoding/json"
	"log"
	"net/http"
	
	"quantum-blockchain/blockchain"
)

type Server struct {
	BC *blockchain.Blockchain
}

func Start(port string, bc *blockchain.Blockchain) {
	s := &Server{BC: bc}
	
	http.HandleFunc("/status", s.StatusHandler)
	http.HandleFunc("/balance", s.BalanceHandler)
	http.HandleFunc("/last-block", s.LastBlockHandler)
	
	log.Printf("🌐 RPC server started on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (s *Server) StatusHandler(w http.ResponseWriter, r *http.Request) {
	lastBlock, err := s.BC.GetLastBlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"last_block_hash": lastBlock.Hash,
		"height":          len(lastBlock.Hash), // В реальном блокчейне здесь будет высота
		"validators":      len(s.BC.Validators),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) BalanceHandler(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Address parameter is required", http.StatusBadRequest)
		return
	}
	
	balance := s.BC.GetBalance([]byte(address))
	response := map[string]string{
		"address": address,
		"balance": balance.String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) LastBlockHandler(w http.ResponseWriter, r *http.Request) {
	block, err := s.BC.GetLastBlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"hash":         block.Hash,
		"timestamp":    block.Timestamp,
		"transactions": len(block.Transactions),
		"validator":    block.Validator,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}