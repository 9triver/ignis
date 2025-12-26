package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/utils"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	mu          sync.Mutex
	host        string
	port        int
	connections map[string]chan *Connection
}

func (m *Manager) wsHandler(w http.ResponseWriter, r *http.Request) {
	connectId := r.Header.Get("executor_id")
	if connectId == "" {
		logrus.Warn("missing connection id in header")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing connection id in header"))
		return
	}

	var upgrader websocket.Upgrader
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("failed upgrading websocket"))
		return
	}
	defer wsConn.Close()

	conn := &Connection{
		name:    connectId,
		send:    make(chan *Message),
		futures: make(map[string]utils.Future[object.Interface]),
	}

	m.mu.Lock()
	if _, ok := m.connections[connectId]; !ok {
		m.connections[connectId] = make(chan *Connection)
	}
	m.connections[connectId] <- conn
	close(m.connections[connectId])
	m.mu.Unlock()

	go func() {
		for msg := range conn.send {
			data, _ := json.Marshal(msg)
			wsConn.WriteMessage(websocket.TextMessage, data)
		}
	}()

	defer conn.Close()
	defer delete(m.connections, connectId)

	for {
		messageType, data, err := wsConn.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.BinaryMessage {
			continue
		}

		if err := conn.handlerMessage(data); err != nil {
			continue
		}
	}
}

func (m *Manager) GetConnection(connId string) *Connection {
	m.mu.Lock()
	if _, ok := m.connections[connId]; !ok {
		m.connections[connId] = make(chan *Connection)
	}
	m.mu.Unlock()

	return <-m.connections[connId]
}

func (m *Manager) Endpoint() string {
	var host string
	if m.host == "0.0.0.0" {
		host = "127.0.0.1"
	} else {
		host = m.host
	}

	return fmt.Sprintf("http://%s:%d/ws", host, m.port)
}

func (m *Manager) Run() error {
	http.HandleFunc("/ws", m.wsHandler)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", m.host, m.port), nil)
}

func NewManager(host string, port int) *Manager {
	m := &Manager{
		host:        host,
		port:        port,
		connections: make(map[string]chan *Connection),
	}

	go m.Run()

	return m
}
