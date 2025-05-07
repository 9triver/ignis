package remote

import (
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
)

type Stub interface {
	SendTo(remoteRef *proto.StoreRef, msg pb.Message)
	RecvChan() <-chan pb.Message
	Close() error
}
