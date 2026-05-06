package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
)

type MessageType string

const (
	MsgOrderStatusUpdate  MessageType = "order_status_update"
	MsgDriverLocation     MessageType = "driver_location"
	MsgOrderAssigned      MessageType = "order_assigned"
	MsgNewOrder           MessageType = "new_order"
	MsgDriverArriving     MessageType = "driver_arriving"
	MsgOrderDelivered     MessageType = "order_delivered"
	MsgPing               MessageType = "ping"
	MsgPong               MessageType = "pong"
)

type Message struct {
	Type      MessageType `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

type Client struct {
	ID       string
	UserID   string
	Role     string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	mu       sync.Mutex
}

type Hub struct {
	clients    map[string]*Client // key: userID
	rooms      map[string]map[string]*Client // key: roomID (e.g. "order:uuid")
	register   chan *Client
	unregister chan *Client
	broadcast  chan *RoomMessage
	rdb        *redis.Client
	mu         sync.RWMutex
}

type RoomMessage struct {
	Room    string
	Message []byte
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]map[string]*Client),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *RoomMessage, 1024),
		rdb:        rdb,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("Client connected: %s (role: %s)", client.UserID, client.Role)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
				// Remove from all rooms
				for roomID, members := range h.rooms {
					delete(members, client.UserID)
					if len(members) == 0 {
						delete(h.rooms, roomID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s", client.UserID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			if members, ok := h.rooms[msg.Room]; ok {
				for _, client := range members {
					select {
					case client.Send <- msg.Message:
					default:
						close(client.Send)
						delete(h.clients, client.UserID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) HandleWebSocket(c *websocket.Conn) {
	userID := c.Params("token") // In production: validate JWT token
	if userID == "" {
		c.Close()
		return
	}

	client := &Client{
		ID:     userID,
		UserID: userID,
		Conn:   c,
		Send:   make(chan []byte, 256),
		Hub:    h,
	}

	h.register <- client

	// Write pump
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer func() {
			ticker.Stop()
			h.unregister <- client
			c.Close()
		}()

		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					c.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				client.mu.Lock()
				err := c.WriteMessage(websocket.TextMessage, msg)
				client.mu.Unlock()
				if err != nil {
					return
				}

			case <-ticker.C:
				client.mu.Lock()
				err := c.WriteMessage(websocket.PingMessage, nil)
				client.mu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}()

	// Read pump
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		var incoming Message
		if err := json.Unmarshal(msg, &incoming); err == nil {
			if incoming.Type == MsgPing {
				pong, _ := json.Marshal(Message{Type: MsgPong, Timestamp: time.Now()})
				client.Send <- pong
			}
		}
	}
}

// JoinRoom adds a client to a room (e.g., order tracking room)
func (h *Hub) JoinRoom(userID, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		if _, ok := h.rooms[roomID]; !ok {
			h.rooms[roomID] = make(map[string]*Client)
		}
		h.rooms[roomID][userID] = client
	}
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(roomID string, msgType MessageType, payload interface{}) {
	msg := Message{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.broadcast <- &RoomMessage{Room: roomID, Message: data}

	// Also publish to Redis for multi-instance support
	if h.rdb != nil {
		ctx := context.Background()
		h.rdb.Publish(ctx, "ws:"+roomID, data)
	}
}

// SendToUser sends a message directly to a specific user
func (h *Hub) SendToUser(userID string, msgType MessageType, payload interface{}) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	msg := Message{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
	}
}

// NotifyOrderUpdate broadcasts order status change to all relevant parties
func (h *Hub) NotifyOrderUpdate(orderID, customerID string, driverID *string, restaurantOwnerID string, payload interface{}) {
	roomID := "order:" + orderID
	h.BroadcastToRoom(roomID, MsgOrderStatusUpdate, payload)
	h.SendToUser(customerID, MsgOrderStatusUpdate, payload)
	h.SendToUser(restaurantOwnerID, MsgOrderStatusUpdate, payload)
	if driverID != nil {
		h.SendToUser(*driverID, MsgOrderStatusUpdate, payload)
	}
}

// UpdateDriverLocation broadcasts driver location to order room
func (h *Hub) UpdateDriverLocation(orderID string, lat, lng float64) {
	h.BroadcastToRoom("order:"+orderID, MsgDriverLocation, map[string]interface{}{
		"order_id":  orderID,
		"latitude":  lat,
		"longitude": lng,
		"timestamp": time.Now(),
	})
}
