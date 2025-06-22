package p2p

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"sync"
	"time"
	
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

const ProtocolID = "/quantum/1.0.0"

type Network struct {
	Host        host.Host
	Peers       map[peer.ID]*peer.AddrInfo
	PeerLock    sync.Mutex
	MessageChan chan Message
}

type Message struct {
	Type    string
	Payload []byte
}

func NewNetwork(ctx context.Context, port int) (*Network, error) {
	// Генерация ключа для ноды
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 2048)
	if err != nil {
		return nil, err
	}

	// Создание P2P хоста
	host, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
	)
	if err != nil {
		return nil, err
	}

	network := &Network{
		Host:        host,
		Peers:       make(map[peer.ID]*peer.AddrInfo),
		MessageChan: make(chan Message, 100),
	}

	// Регистрация обработчика входящих соединений
	host.SetStreamHandler(ProtocolID, network.handleStream)

	log.Printf("🛰️ P2P Node ID: %s", host.ID())
	log.Printf("📡 Listening on: %v", host.Addrs())

	// Автообновление списка пиров
	go network.discoverPeers(ctx)

	return network, nil
}

func (n *Network) handleStream(s network.Stream) {
	defer s.Close()
	
	var msg Message
	decoder := gob.NewDecoder(s)
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Stream decode error: %v", err)
		return
	}

	log.Printf("📨 Received %s message from %s", msg.Type, s.Conn().RemotePeer())
	n.MessageChan <- msg
}

func (n *Network) discoverPeers(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.PeerLock.Lock()
			log.Printf("🌐 Known peers: %d", len(n.Peers))
			n.PeerLock.Unlock()
		}
	}
}

func (n *Network) Connect(ctx context.Context, peerAddr string) error {
	// Парсинг адреса пира
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return err
	}

	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	// Добавление в пирсторе
	n.Host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, peerstore.PermanentAddrTTL)

	// Установка соединения
	if err := n.Host.Connect(ctx, *addrInfo); err != nil {
		return err
	}

	n.PeerLock.Lock()
	defer n.PeerLock.Unlock()
	n.Peers[addrInfo.ID] = addrInfo

	log.Printf("✅ Connected to: %s", addrInfo.ID)
	return nil
}

func (n *Network) Broadcast(msgType string, payload []byte) {
	msg := Message{Type: msgType, Payload: payload}
	
	n.PeerLock.Lock()
	defer n.PeerLock.Unlock()
	
	for peerID := range n.Peers {
		go n.sendToPeer(peerID, msg)
	}
}

func (n *Network) sendToPeer(peerID peer.ID, msg Message) {
	// Открытие стрима
	s, err := n.Host.NewStream(context.Background(), peerID, ProtocolID)
	if err != nil {
		log.Printf("Failed to open stream to %s: %v", peerID, err)
		delete(n.Peers, peerID)
		return
	}
	defer s.Close()
	
	// Отправка сообщения
	encoder := gob.NewEncoder(s)
	if err := encoder.Encode(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}