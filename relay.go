package main

import (
	"io"
	"log"
	"net"
	"sync"
)

// Relay manages TCP -> UDP forwarding
type Relay struct {
	ListenAddr string
	UDPConn    *net.UDPConn
	ClientAddr *net.UDPAddr

	mu      sync.Mutex
	conns   map[uint32]net.Conn
	nextCID uint32
}

func NewRelay(listenAddr, clientAddr string) (*Relay, error) {
	// prepare UDP tunnel socket
	udpAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	cAddr, err := net.ResolveUDPAddr("udp", clientAddr)
	if err != nil {
		return nil, err
	}

	return &Relay{
		ListenAddr: listenAddr,
		UDPConn:    udpConn,
		ClientAddr: cAddr,
		conns:      make(map[uint32]net.Conn),
		nextCID:    1,
	}, nil
}

func (r *Relay) Run() error {
	// Start UDP reader (to handle responses from client)
	go r.readFromTunnel()

	// Start TCP listener for incoming connections
	ln, err := net.Listen("tcp", r.ListenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("Relay listening on %s, tunneling to %s\n", r.ListenAddr, r.ClientAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		r.handleTCP(conn)
	}
}

func (r *Relay) handleTCP(conn net.Conn) {
	r.mu.Lock()
	cid := r.nextCID
	r.nextCID++
	r.conns[cid] = conn
	r.mu.Unlock()

	log.Printf("New TCP conn %d from %s\n", cid, conn.RemoteAddr())

	// Notify client (Frame NEW)
	newFrame := &Frame{Version: FrameVersion, Type: FrameTypeNew, ConnID: cid, Payload: []byte{}}
	data, _ := newFrame.Encode()
	r.UDPConn.WriteToUDP(data, r.ClientAddr)

	// Stream TCP -> UDP
	go func() {
		defer conn.Close()
		defer r.closeConn(cid)

		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Println("TCP read error:", err)
				}
				return
			}
			frame := &Frame{Version: FrameVersion, Type: FrameTypeData, ConnID: cid, Payload: buf[:n]}
			data, _ := frame.Encode()
			r.UDPConn.WriteToUDP(data, r.ClientAddr)
		}
	}()
}

func (r *Relay) readFromTunnel() {
	buf := make([]byte, 1500)
	for {
		n, addr, err := r.UDPConn.ReadFromUDP(buf)
		if err != nil {
			log.Println("UDP read error:", err)
			continue
		}
		if addr.String() != r.ClientAddr.String() {
			log.Println("Ignoring packet from unknown addr:", addr)
			continue
		}
		frame, err := DecodeFrame(buf[:n])
		if err != nil {
			log.Println("frame decode error:", err)
			continue
		}
		r.handleFrame(frame)
	}
}

func (r *Relay) handleFrame(f *Frame) {
	switch f.Type {
	case FrameTypeData:
		r.mu.Lock()
		conn := r.conns[f.ConnID]
		r.mu.Unlock()
		if conn != nil {
			conn.Write(f.Payload)
		}
	case FrameTypeFin:
		r.closeConn(f.ConnID)
	default:
		log.Printf("Unhandled frame type %d\n", f.Type)
	}
}

func (r *Relay) closeConn(cid uint32) {
	r.mu.Lock()
	conn := r.conns[cid]
	delete(r.conns, cid)
	r.mu.Unlock()

	if conn != nil {
		conn.Close()
		log.Printf("Closed TCP conn %d\n", cid)
	}

	// Send FIN to client
	fin := &Frame{Version: FrameVersion, Type: FrameTypeFin, ConnID: cid}
	data, _ := fin.Encode()
	r.UDPConn.WriteToUDP(data, r.ClientAddr)
}
