package configs

import (
	"os"
	"path"
	"time"

	"github.com/9triver/ignis/utils/errors"
)

const (
	kDefaultSystemTimeout     = 300 * time.Second
	kDefaultExecutionTimeout  = 300 * time.Second
	kDefaultChannelBufferSize = 16

	AppName = "actor-platform"
)

var (
	FlowTimeout       = kDefaultSystemTimeout    // flow object requests
	ExecutionTimeout  = kDefaultExecutionTimeout // actor execution
	ChannelBufferSize = kDefaultChannelBufferSize
)

var StoragePath = func() string {
	home, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	dir := path.Join(home, AppName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(errors.WrapWith(err, "error starting platform"))
	}
	return dir
}()
