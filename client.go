package main

import (
	"io"
	"log"
	"net"
	"sync"
)

// Client manages UDP -> TCP forwarding
type Client struct {
	ListenUDP  *net.UDPConn
	RelayAddr  *net.UDPAddr
	DestAddr   string // where to connect for new sessions

	mu    sync.Mutex
	conns map[uint32]net.Conn
}

func NewClient(listenAddr, relayAddr, destAddr string) (*Client, error) {
	// UDP listen socket
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	rAddr, err := net.ResolveUDPAddr("udp", relayAddr)
	if err != nil {
		return nil, err
	}

	return &Client{
		ListenUDP: udpConn,
		RelayAddr: rAddr,
		DestAddr:  destAddr,
		conns:     make(map[uint32]net.Conn),
	}, nil
}

func (c *Client) Run() {
	buf := make([]byte, 1500)
	for {
		n, addr, err := c.ListenUDP.ReadFromUDP(buf)
		if err != nil {
			log.Println("UDP read error:", err)
			continue
		}
		if addr.String() != c.RelayAddr.String() {
			log.Println("Ignoring packet from unknown addr:", addr)
			continue
		}
		frame, err := DecodeFrame(buf[:n])
		if err != nil {
			log.Println("frame decode error:", err)
			continue
		}
		c.handleFrame(frame)
	}
}

func (c *Client) handleFrame(f *Frame) {
	switch f.Type {
	case FrameTypeNew:
		c.openConn(f.ConnID)
	case FrameTypeData:
		c.mu.Lock()
		conn := c.conns[f.ConnID]
		c.mu.Unlock()
		if conn != nil {
			conn.Write(f.Payload)
		}
	case FrameTypeFin:
		c.closeConn(f.ConnID)
	default:
		log.Printf("Unhandled frame type %d\n", f.Type)
	}
}

func (c *Client) openConn(cid uint32) {
	conn, err := net.Dial("tcp", c.DestAddr)
	if err != nil {
		log.Printf("Failed to open TCP %s for ConnID %d: %v", c.DestAddr, cid, err)
		// send FIN back
		fin := &Frame{Version: FrameVersion, Type: FrameTypeFin, ConnID: cid}
		data, _ := fin.Encode()
		c.ListenUDP.WriteToUDP(data, c.RelayAddr)
		return
	}

	c.mu.Lock()
	c.conns[cid] = conn
	c.mu.Unlock()

	log.Printf("Opened TCP to %s for ConnID %d\n", c.DestAddr, cid)

	// Pump TCP -> UDP
	go func() {
		defer c.closeConn(cid)
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
			c.ListenUDP.WriteToUDP(data, c.RelayAddr)
		}
	}()
}

func (c *Client) closeConn(cid uint32) {
	c.mu.Lock()
	conn := c.conns[cid]
	delete(c.conns, cid)
	c.mu.Unlock()

	if conn != nil {
		conn.Close()
		log.Printf("Closed TCP conn %d\n", cid)
	}

	// send FIN to relay
	fin := &Frame{Version: FrameVersion, Type: FrameTypeFin, ConnID: cid}
	data, _ := fin.Encode()
	c.ListenUDP.WriteToUDP(data, c.RelayAddr)
}
