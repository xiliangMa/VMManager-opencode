package websocket

import (
	"encoding/json"
	"net/http"

	"vmmanager/internal/libvirt"

	"github.com/gorilla/websocket"
)

type Handler struct {
	upgrader websocket.Upgrader
	clients  map[string]*Client
	libvirt  *libvirt.Client
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
	vmID string
}

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func NewHandler(libvirtClient *libvirt.Client) *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients: make(map[string]*Client),
		libvirt: libvirtClient,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vmID := r.URL.Query().Get("vm_id")
	token := r.URL.Query().Get("token")

	if vmID == "" || token == "" {
		http.Error(w, "vm_id and token are required", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
		vmID: vmID,
	}

	h.clients[vmID] = client

	go client.writePump()
	go client.readPump(h)
}

func (c *Client) readPump(h *Handler) {
	defer func() {
		delete(h.clients, c.vmID)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(nil)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(nil)
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "resize":
			handleResize(c, msg.Payload)
		case "key":
			handleKey(c, msg.Payload)
		case "mouse":
			handleMouse(c, msg.Payload)
		}
	}
}

func (c *Client) writePump() {
	ticker := *ticker
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(nil)
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(nil)
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func handleResize(c *Client, payload json.RawMessage) {}

func handleKey(c *Client, payload json.RawMessage) {}

func handleMouse(c *Client, payload json.RawMessage) {}
