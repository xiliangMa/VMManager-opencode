package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"vmmanager/internal/libvirt"

	"github.com/gorilla/websocket"
)

type Handler struct {
	upgrader websocket.Upgrader
	clients  map[string]*VNCClient
	libvirt  *libvirt.Client
}

type VNCClient struct {
	conn       *websocket.Conn
	vmID       string
	send       chan []byte
	recv       chan []byte
	closed     bool
	connMu     sync.Mutex
	targetConn net.Conn
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
	Host         string `json:"host"`
	Port         int    `json:"port"`
	WebSocketURL string `json:"websocket_url"`
}

func NewHandler(libvirtClient *libvirt.Client) *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients: make(map[string]*VNCClient),
		libvirt: libvirtClient,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[VNC] WebSocket request: path=%s", r.URL.Path)

	vmID := r.URL.Query().Get("vm_id")
	if vmID == "" {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if strings.HasPrefix(path, "ws/vnc/") {
			vmID = strings.TrimPrefix(path, "ws/vnc/")
		} else if strings.HasPrefix(path, "api/v1/ws/vnc/") {
			vmID = strings.TrimPrefix(path, "api/v1/ws/vnc/")
		} else if strings.HasPrefix(path, "ws/") {
			vmID = strings.TrimPrefix(path, "ws/")
		}
	}

	log.Printf("[VNC] VM ID: %s", vmID)

	if vmID == "" {
		log.Printf("[VNC] VM ID is empty")
		http.Error(w, "vm_id is required", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[VNC] Failed to upgrade: %v", err)
		return
	}

	client := &VNCClient{
		conn: conn,
		vmID: vmID,
		send: make(chan []byte, 1024),
		recv: make(chan []byte, 1024),
	}

	h.clients[vmID] = client

	log.Printf("[VNC] Starting VNC proxy for: %s", vmID)

	go client.proxyVNC(h)
	go client.writePump()
	go client.readPump()
}

func extractVNCPort(xmlDesc string) (int, error) {
	for _, line := range strings.Split(xmlDesc, "\n") {
		if strings.Contains(line, "<graphics") && strings.Contains(line, "type='vnc'") {
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "port='") {
					portStr := strings.TrimPrefix(part, "port='")
					portStr = strings.TrimSuffix(portStr, "'")
					var port int
					if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
						return port, nil
					}
				}
			}
		}
	}
	return 5900, nil
}

func (c *VNCClient) proxyVNC(h *Handler) {
	log.Printf("[VNC] Starting proxy for: %s", c.vmID)

	if h.libvirt == nil || !h.libvirt.IsConnected() {
		log.Printf("[VNC] libvirt not connected")
		return
	}

	domain, err := h.libvirt.LookupByUUID(c.vmID)
	if err != nil {
		log.Printf("[VNC] Domain not found: %v", err)
		return
	}

	state, _, err := domain.GetState()
	if err != nil {
		log.Printf("[VNC] Failed to get domain state: %v", err)
		return
	}

	if state != 1 {
		log.Printf("[VNC] Domain is not running, state: %d", state)
		return
	}

	xmlDesc, err := domain.GetXMLDesc()
	if err != nil {
		log.Printf("[VNC] Failed to get domain XML: %v", err)
		return
	}

	vncPort, err := extractVNCPort(xmlDesc)
	if err != nil {
		log.Printf("[VNC] Failed to extract VNC port: %v", err)
		return
	}

	log.Printf("[VNC] Connecting to QEMU VNC port: %d", vncPort)

	addr := fmt.Sprintf("127.0.0.1:%d", vncPort)
	c.targetConn, err = net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("[VNC] Failed to connect to VNC: %v", err)
		return
	}

	log.Printf("[VNC] Connected to QEMU VNC")

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := c.targetConn.Read(buf)
			if err != nil {
				log.Printf("[VNC] VNC read error: %v", err)
				return
			}
			select {
			case c.send <- buf[:n]:
			default:
			}
		}
	}()

	go func() {
		for msg := range c.recv {
			if c.targetConn != nil {
				c.targetConn.Write(msg)
			}
		}
	}()
}

func (c *VNCClient) readPump() {
	defer func() {
		c.connMu.Lock()
		if !c.closed {
			c.closed = true
			c.conn.Close()
		}
		if c.targetConn != nil {
			c.targetConn.Close()
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

func (h *Handler) GetVNCConsoleURL(vmID string) string {
	return fmt.Sprintf("/ws/vnc/%s", vmID)
}

func (h *Handler) GetConsoleInfo(vmID string) (*ConsoleInfo, error) {
	if h.libvirt == nil || !h.libvirt.IsConnected() {
		return &ConsoleInfo{
			Host: "127.0.0.1",
			Port: 5900,
		}, nil
	}

	domain, err := h.libvirt.LookupByUUID(vmID)
	if err != nil {
		return nil, fmt.Errorf("VM not found: %s", vmID)
	}

	state, _, err := domain.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain state: %w", err)
	}

	if state != 1 {
		return nil, fmt.Errorf("VM is not running")
	}

	xmlDesc, err := domain.GetXMLDesc()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain XML: %w", err)
	}

	port, err := extractVNCPort(xmlDesc)
	if err != nil {
		return nil, fmt.Errorf("failed to extract VNC port: %w", err)
	}

	return &ConsoleInfo{
		Host:         "127.0.0.1",
		Port:         port,
		WebSocketURL: fmt.Sprintf("/ws/vnc/%s", vmID),
	}, nil
}
