package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"vmmanager/internal/libvirt"
)

type VNCClient struct {
	conn     *websocket.Conn
	vmID     string
	send     chan []byte
	recv     chan []byte
	closed   bool
	connMu   sync.Mutex
	vmClient *libvirt.Client
}

type VNCPayload struct {
	Width  int  `json:"width"`
	Height int  `json:"height"`
	X      int  `json:"x"`
	Y      int  `json:"y"`
	Button int  `json:"button"`
	Key    int  `json:"key"`
	Down   bool `json:"down"`
}

type VNCMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type ConsoleInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

var vncUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) HandleVNC(w http.ResponseWriter, r *http.Request, vmID string) {
	log.Printf("[VNC] HandleVNC called: vmID=%s, path=%s", vmID, r.URL.Path)

	if vmID == "" {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if strings.HasPrefix(path, "ws/vnc/") {
			vmID = strings.TrimPrefix(path, "ws/vnc/")
		} else if strings.HasPrefix(path, "api/v1/ws/vnc/") {
			vmID = strings.TrimPrefix(path, "api/v1/ws/vnc/")
		}
	}

	log.Printf("[VNC] Final VM ID: %s", vmID)

	conn, err := vncUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[VNC] Failed to upgrade: %v", err)
		return
	}

	client := &VNCClient{
		conn:     conn,
		vmID:     vmID,
		send:     make(chan []byte, 1024),
		recv:     make(chan []byte, 1024),
		vmClient: h.libvirt,
	}

	log.Printf("[VNC] Starting VNC proxy for: %s", vmID)

	go client.writePump()
	go client.readPump()
	go client.proxyVNC()
}

func (c *VNCClient) readPump() {
	defer func() {
		c.connMu.Lock()
		if !c.closed {
			c.closed = true
			c.conn.Close()
		}
		c.connMu.Unlock()
	}()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg VNCMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "mouse":
			c.handleMouse(msg.Payload)
		case "keyboard":
			c.handleKeyboard(msg.Payload)
		case "resize":
			c.handleResize(msg.Payload)
		}
	}
}

func (c *VNCClient) writePump() {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		c.connMu.Lock()
		if !c.closed {
			c.closed = true
			c.conn.Close()
		}
		c.connMu.Unlock()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.connMu.Lock()
			if c.closed {
				c.connMu.Unlock()
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.connMu.Unlock()
				return
			}

			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				c.connMu.Unlock()
				return
			}
			c.connMu.Unlock()

		case <-ticker.C:
			c.connMu.Lock()
			if c.closed {
				c.connMu.Unlock()
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.connMu.Unlock()
				return
			}
			c.connMu.Unlock()
		}
	}
}

func (c *VNCClient) proxyVNC() {
	log.Printf("[VNC] Proxy starting for VM: %s", c.vmID)

	c.send <- []byte(`{"type":"connected","payload":{"message":"VNC connection ready (mock mode)"}}`)
	log.Printf("[VNC] Sent connected message to client")

	go func() {
		for {
			select {
			case msg := <-c.recv:
				log.Printf("[VNC] Received message from client: %v", msg)
			case <-time.After(30 * time.Second):
				log.Printf("[VNC] No activity, keeping connection alive")
			}
		}
	}()
}

func (c *VNCClient) handleResize(payload json.RawMessage) {
	var resize VNCPayload
	if err := json.Unmarshal(payload, &resize); err != nil {
		return
	}

	msg := []byte{0x04, 0x02}
	msg = append(msg, byte((resize.Width>>8)&0xFF))
	msg = append(msg, byte(resize.Width&0xFF))
	msg = append(msg, byte((resize.Height>>8)&0xFF))
	msg = append(msg, byte(resize.Height&0xFF))

	select {
	case c.recv <- msg:
	default:
	}
}

func (c *VNCClient) handleKeyboard(payload json.RawMessage) {
	var key VNCPayload
	if err := json.Unmarshal(payload, &key); err != nil {
		return
	}

	var msg []byte
	if key.Down {
		msg = []byte{0x01, byte((key.Key >> 8) & 0xFF), byte(key.Key & 0xFF)}
	} else {
		msg = []byte{0x01, byte(((key.Key >> 8) & 0xFF) | 0x80), byte(key.Key & 0xFF)}
	}

	select {
	case c.recv <- msg:
	default:
	}
}

func (c *VNCClient) handleMouse(payload json.RawMessage) {
	var mouse VNCPayload
	if err := json.Unmarshal(payload, &mouse); err != nil {
		return
	}

	msg := []byte{0x05, byte(mouse.Button)}
	msg = append(msg, byte((mouse.X>>8)&0xFF))
	msg = append(msg, byte(mouse.X&0xFF))
	msg = append(msg, byte((mouse.Y>>8)&0xFF))
	msg = append(msg, byte(mouse.Y&0xFF))

	select {
	case c.recv <- msg:
	default:
	}
}

func (h *Handler) GetVNCConsoleURL(vmID string) (string, error) {
	return fmt.Sprintf("/ws/vnc/%s", vmID), nil
}

func (h *Handler) GetConsoleInfo(vmID string) (*ConsoleInfo, error) {
	vm := h.libvirt.Domains[vmID]
	if vm == nil {
		return nil, fmt.Errorf("VM not found: %s", vmID)
	}

	port := vm.VNCPort
	if port == 0 {
		port = 5900
	}

	return &ConsoleInfo{
		Host: "127.0.0.1",
		Port: port,
	}, nil
}
