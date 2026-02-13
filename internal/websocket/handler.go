package websocket

import (
	"crypto/des"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
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
	log.Printf("[VNC][%s] Starting proxy", c.vmID)

	if h.libvirt == nil {
		log.Printf("[VNC][%s] libvirt is nil", c.vmID)
		return
	}

	if !h.libvirt.IsConnected() {
		log.Printf("[VNC][%s] libvirt not connected", c.vmID)
		return
	}

	log.Printf("[VNC][%s] Looking up domain", c.vmID)
	domain, err := h.libvirt.LookupByUUID(c.vmID)
	if err != nil {
		log.Printf("[VNC][%s] Domain not found: %v", c.vmID, err)
		return
	}

	state, _, err := domain.GetState()
	if err != nil {
		log.Printf("[VNC][%s] Failed to get domain state: %v", c.vmID, err)
		return
	}

	log.Printf("[VNC][%s] Domain state: %d", c.vmID, state)
	if state != 1 {
		log.Printf("[VNC][%s] Domain not running", c.vmID)
		return
	}

	xmlDesc, err := domain.GetXMLDesc()
	if err != nil {
		log.Printf("[VNC][%s] Failed to get domain XML: %v", c.vmID, err)
		return
	}

	vncPort, err := extractVNCPort(xmlDesc)
	if err != nil {
		log.Printf("[VNC][%s] Failed to extract VNC port: %v", c.vmID, err)
		return
	}

	log.Printf("[VNC][%s] VNC port: %d, connecting...", c.vmID, vncPort)

	addr := fmt.Sprintf("127.0.0.1:%d", vncPort)
	c.targetConn, err = net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("[VNC][%s] Failed to connect to VNC: %v", c.vmID, err)
		return
	}

	log.Printf("[VNC][%s] Connected to QEMU VNC, starting transparent proxy", c.vmID)

	// Note: We don't perform VNC handshake here.
	// The WebSocket proxy should be transparent, forwarding raw bytes between
	// the noVNC client and the VNC server. The noVNC client will handle the
	// VNC protocol handshake (version, auth, init) directly with the server.

	go c.vncToWS()
	go c.wsToVNC()
}

// performVNCHandshake completes the VNC protocol handshake and authentication
func (c *VNCClient) performVNCHandshake(h *Handler) error {
	// Read server version (12 bytes)
	buf := make([]byte, 12)
	if _, err := io.ReadFull(c.targetConn, buf); err != nil {
		return fmt.Errorf("failed to read server version: %w", err)
	}
	log.Printf("[VNC][%s] Server version: %s", c.vmID, string(buf))

	// Send client version (RFB 003.008)
	if _, err := c.targetConn.Write([]byte("RFB 003.008\n")); err != nil {
		return fmt.Errorf("failed to send client version: %w", err)
	}

	// Read number of security types
	var nSecTypes uint8
	if err := binary.Read(c.targetConn, binary.BigEndian, &nSecTypes); err != nil {
		return fmt.Errorf("failed to read security types count: %w", err)
	}
	log.Printf("[VNC][%s] Security types count: %d", c.vmID, nSecTypes)

	if nSecTypes == 0 {
		return fmt.Errorf("no security types offered")
	}

	// Read security types
	secTypes := make([]uint8, nSecTypes)
	if _, err := io.ReadFull(c.targetConn, secTypes); err != nil {
		return fmt.Errorf("failed to read security types: %w", err)
	}
	log.Printf("[VNC][%s] Security types: %v", c.vmID, secTypes)

	// Select security type (prefer None=1, otherwise VNC auth=2)
	var selectedType uint8 = 0xFF
	for _, t := range secTypes {
		if t == 1 { // None authentication
			selectedType = 1
			break
		} else if t == 2 && selectedType == 0xFF { // VNC authentication
			selectedType = 2
		}
	}

	if selectedType == 0xFF {
		return fmt.Errorf("no supported security type found")
	}

	// Send selected security type
	if err := binary.Write(c.targetConn, binary.BigEndian, selectedType); err != nil {
		return fmt.Errorf("failed to send security type: %w", err)
	}
	log.Printf("[VNC][%s] Selected security type: %d", c.vmID, selectedType)

	// Handle VNC authentication (type 2)
	if selectedType == 2 {
		// Read challenge (16 bytes)
		challenge := make([]byte, 16)
		if _, err := io.ReadFull(c.targetConn, challenge); err != nil {
			return fmt.Errorf("failed to read challenge: %w", err)
		}

		// Get VNC password from libvirt domain
		password := c.getVNCPassword(h)
		response := c.encryptVNCChallenge(challenge, password)
		
		if _, err := c.targetConn.Write(response); err != nil {
			return fmt.Errorf("failed to send auth response: %w", err)
		}
		log.Printf("[VNC][%s] Sent password response (password length: %d)", c.vmID, len(password))
	}

	// Read security result
	var secResult uint32
	if err := binary.Read(c.targetConn, binary.BigEndian, &secResult); err != nil {
		return fmt.Errorf("failed to read security result: %w", err)
	}
	log.Printf("[VNC][%s] Security result: %d", c.vmID, secResult)

	if secResult != 0 {
		return fmt.Errorf("authentication failed: %d", secResult)
	}

	// Send ClientInit (shared flag = 1)
	if err := binary.Write(c.targetConn, binary.BigEndian, uint8(1)); err != nil {
		return fmt.Errorf("failed to send client init: %w", err)
	}

	// Read ServerInit
	var width, height uint16
	if err := binary.Read(c.targetConn, binary.BigEndian, &width); err != nil {
		return fmt.Errorf("failed to read server init width: %w", err)
	}
	if err := binary.Read(c.targetConn, binary.BigEndian, &height); err != nil {
		return fmt.Errorf("failed to read server init height: %w", err)
	}

	// Read pixel format (16 bytes)
	pixelFormat := make([]byte, 16)
	if _, err := io.ReadFull(c.targetConn, pixelFormat); err != nil {
		return fmt.Errorf("failed to read pixel format: %w", err)
	}

	// Read desktop name length
	var nameLen uint32
	if err := binary.Read(c.targetConn, binary.BigEndian, &nameLen); err != nil {
		return fmt.Errorf("failed to read desktop name length: %w", err)
	}

	// Read desktop name
	desktopName := make([]byte, nameLen)
	if _, err := io.ReadFull(c.targetConn, desktopName); err != nil {
		return fmt.Errorf("failed to read desktop name: %w", err)
	}

	log.Printf("[VNC][%s] Handshake complete: %dx%d - %s", c.vmID, width, height, string(desktopName))
	return nil
}

// getVNCPassword retrieves the VNC password from the domain XML
func (c *VNCClient) getVNCPassword(h *Handler) string {
	domain, err := h.libvirt.LookupByUUID(c.vmID)
	if err != nil {
		return ""
	}
	
	xmlDesc, err := domain.GetXMLDesc()
	if err != nil {
		return ""
	}
	
	// Extract passwd attribute from graphics element
	for _, line := range strings.Split(xmlDesc, "\n") {
		if strings.Contains(line, "<graphics") && strings.Contains(line, "type='vnc'") {
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "passwd='") {
					passwd := strings.TrimPrefix(part, "passwd='")
					passwd = strings.TrimSuffix(passwd, "'")
					return passwd
				}
			}
		}
	}
	return ""
}

// encryptVNCChallenge encrypts the challenge using VNC DES algorithm
func (c *VNCClient) encryptVNCChallenge(challenge []byte, password string) []byte {
	// VNC uses a modified DES with reversed key bits
	// Pad password to 8 bytes
	key := make([]byte, 8)
	copy(key, password)
	
	// Reverse bits in each byte (VNC specific)
	for i := range key {
		key[i] = reverseBits(key[i])
	}
	
	// Encrypt challenge (16 bytes = 2 blocks)
	response := make([]byte, 16)
	
	block, err := des.NewCipher(key)
	if err != nil {
		// Fallback: return empty response
		return response
	}
	
	// Encrypt first 8 bytes
	block.Encrypt(response[0:8], challenge[0:8])
	// Encrypt second 8 bytes
	block.Encrypt(response[8:16], challenge[8:16])
	
	return response
}

// reverseBits reverses the bits in a byte
func reverseBits(b byte) byte {
	var result byte
	for i := 0; i < 8; i++ {
		result = (result << 1) | ((b >> i) & 1)
	}
	return result
}

func (c *VNCClient) vncToWS() {
	buf := make([]byte, 4096)
	for {
		c.connMu.Lock()
		if c.closed || c.targetConn == nil {
			c.connMu.Unlock()
			return
		}
		c.connMu.Unlock()

		n, err := c.targetConn.Read(buf)
		if err != nil {
			log.Printf("[VNC][%s] VNC read error: %v", c.vmID, err)
			return
		}

		log.Printf("[VNC][%s] Read %d bytes from VNC", c.vmID, n)

		c.connMu.Lock()
		if c.closed {
			c.connMu.Unlock()
			return
		}
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
			log.Printf("[VNC][%s] WS write error: %v", c.vmID, err)
			c.connMu.Unlock()
			return
		}
		log.Printf("[VNC][%s] Wrote %d bytes to WS", c.vmID, n)
		c.connMu.Unlock()
	}
}

func (c *VNCClient) wsToVNC() {
	for msg := range c.recv {
		c.connMu.Lock()
		if c.closed || c.targetConn == nil {
			c.connMu.Unlock()
			return
		}
		c.connMu.Unlock()

		if _, err := c.targetConn.Write(msg); err != nil {
			log.Printf("[VNC][%s] VNC write error: %v", c.vmID, err)
			return
		}
	}
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
			log.Printf("[VNC][%s] WebSocket read error: %v", c.vmID, err)
			break
		}

		log.Printf("[VNC][%s] Received %d bytes, type: %d", c.vmID, len(message), message[0])

		// Try to parse as JSON message (for mouse/keyboard/resize events)
		var msg VNCMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			// JSON message - handle control commands
			log.Printf("[VNC][%s] Received JSON message type: %s", c.vmID, msg.Type)
			switch msg.Type {
			case "mouse":
				c.handleMouse(msg.Payload)
			case "keyboard":
				c.handleKeyboard(msg.Payload)
			case "resize":
				c.handleResize(msg.Payload)
			}
		} else {
			// Binary message - forward directly to VNC server
			// This includes VNC protocol handshake and framebuffer requests
			c.connMu.Lock()
			if c.closed || c.targetConn == nil {
				c.connMu.Unlock()
				return
			}
			c.connMu.Unlock()

			if _, err := c.targetConn.Write(message); err != nil {
				log.Printf("[VNC][%s] VNC write error: %v", c.vmID, err)
				return
			}
			log.Printf("[VNC][%s] Forwarded %d bytes to VNC", c.vmID, len(message))
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
