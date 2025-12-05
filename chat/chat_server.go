package chat

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Message represents a chat message
type Message struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Connection represents an active client connection
type Connection struct {
	SessionID    string
	GitHubUser   string
	MessageChan  chan Message
	LastActivity time.Time
}

// Server manages chat connections and message broadcasting
type Server struct {
	connections sync.Map // sessionID -> *Connection
	messages    []Message
	messagesMux sync.RWMutex
	maxMessages int
}

// NewServer creates a new chat server
func NewServer() *Server {
	return &Server{
		messages:    make([]Message, 0),
		maxMessages: 100, // Keep last 100 messages
	}
}

// RegisterConnection adds a new client connection
func (s *Server) RegisterConnection(sessionID, githubUser string) *Connection {
	conn := &Connection{
		SessionID:    sessionID,
		GitHubUser:   githubUser,
		MessageChan:  make(chan Message, 10),
		LastActivity: time.Now(),
	}
	s.connections.Store(sessionID, conn)
	
	// Send join notification
	s.BroadcastSystemMessage(fmt.Sprintf("%s joined the chat", githubUser))
	
	return conn
}

// UnregisterConnection removes a client connection
func (s *Server) UnregisterConnection(sessionID string) {
	if connInterface, ok := s.connections.LoadAndDelete(sessionID); ok {
		conn := connInterface.(*Connection)
		close(conn.MessageChan)
		
		// Send leave notification
		s.BroadcastSystemMessage(fmt.Sprintf("%s left the chat", conn.GitHubUser))
	}
}

// GetConnection retrieves a connection by session ID
func (s *Server) GetConnection(sessionID string) (*Connection, bool) {
	if connInterface, ok := s.connections.Load(sessionID); ok {
		return connInterface.(*Connection), true
	}
	return nil, false
}

// BroadcastMessage sends a message to all connected clients
func (s *Server) BroadcastMessage(sender, message string) error {
	msg := Message{
		ID:        generateMessageID(),
		Sender:    sender,
		Message:   message,
		Timestamp: time.Now(),
	}
	
	// Store message in history
	s.messagesMux.Lock()
	s.messages = append(s.messages, msg)
	if len(s.messages) > s.maxMessages {
		s.messages = s.messages[1:] // Remove oldest message
	}
	s.messagesMux.Unlock()
	
	// Broadcast to all connections
	s.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		select {
		case conn.MessageChan <- msg:
			// Message sent successfully
		default:
			// Channel full, skip this client
		}
		return true
	})
	
	return nil
}

// BroadcastSystemMessage sends a system notification
func (s *Server) BroadcastSystemMessage(message string) {
	s.BroadcastMessage("System", message)
}

// GetMessageHistory returns recent messages
func (s *Server) GetMessageHistory(limit int) []Message {
	s.messagesMux.RLock()
	defer s.messagesMux.RUnlock()
	
	if limit <= 0 || limit > len(s.messages) {
		limit = len(s.messages)
	}
	
	start := len(s.messages) - limit
	if start < 0 {
		start = 0
	}
	
	history := make([]Message, limit)
	copy(history, s.messages[start:])
	return history
}

// GetActiveUsers returns list of currently connected users
func (s *Server) GetActiveUsers() []string {
	users := make([]string, 0)
	s.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		users = append(users, conn.GitHubUser)
		return true
	})
	return users
}

// MessageToJSON converts a message to JSON string
func MessageToJSON(msg Message) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// generateMessageID creates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
