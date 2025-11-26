package sender

// Sender interface
type Sender interface {
	Send(dst string, data []byte)
	Listen(func(msg []byte))
	Accept()
	GetConnectionType(dst string) (typeString string, ok bool)
	Close()
}
