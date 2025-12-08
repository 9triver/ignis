package remote

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/utils"
)

type Message struct {
	CorrId string `json:"corr_id"`
	Topic  string `json:"topic"`
	Value  any    `json:"value,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Connection struct {
	name string
	mu   sync.Mutex

	send    chan *Message
	futures map[string]utils.Future[object.Interface]
}

func (m *Connection) handlerMessage(data []byte) error {
	msg := &Message{}
	if err := json.Unmarshal(data, msg); err != nil {
		return err
	}

	fut, ok := m.futures[msg.CorrId]
	if !ok {
		return errors.New("no such future")
	}

	fut.Resolve(object.NewLocal(msg.Value, object.LangJson))
	return nil
}

func (m *Connection) Execute(topic string, args map[string]object.Interface) utils.Future[object.Interface] {
	fut := utils.NewFuture[object.Interface](configs.ExecutionTimeout)
	values := make(map[string]any)

	// 编码所有参数
	for param, obj := range args {
		value, err := obj.Value()
		if err != nil {
			fut.Reject(err)
			return fut
		}
		values[param] = value
	}

	m.mu.Lock()
	corrId := utils.GenID()
	m.futures[corrId] = fut
	m.mu.Unlock()

	fut.OnDone(func(_ object.Interface, _ time.Duration, _ error) {
		m.mu.Lock()
		delete(m.futures, corrId)
		m.mu.Unlock()
	})

	msg := &Message{
		CorrId: corrId,
		Topic:  topic,
		Value:  values,
	}

	m.send <- msg

	return fut
}

func (m *Connection) Close() {
	close(m.send)
}
