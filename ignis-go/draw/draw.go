package draw

import (
	"encoding/json"
	"fmt"
	"sync"

	"gopkg.in/zeromq/goczmq.v4"
)

var (
	sock *goczmq.Sock
	once sync.Once
	err  error
)

func InitSock() {
	once.Do(func() {
		sock, err = goczmq.NewPush("tcp://localhost:6666")
		if err != nil {
			fmt.Printf("Error creating socket: %v\n", err)
			return
		}
	})
}

func SendMessage(module, direction string) error {
	InitSock()

	message := map[string]interface{}{
		"component": "CentralDispatcher",
		"msg": map[string]string{
			"module":    module,
			"direction": direction,
		},
	}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	fmt.Printf("Sent message: %s (%s)\n", module, direction)
	// Send JSON as a single frame
	if err := sock.SendFrame(data, goczmq.FlagDontWait); err != nil {
		return err
	}
	return nil
}
