package configs

import (
	"github.com/9triver/ignis/utils/errors"
	"os"
	"path"
	"time"
)

const (
	// Timeouts for platform futures
	kDefaultSystemTimeout    = 30 * time.Second
	kDefaultExecutionTimeout = 300 * time.Second

	// Values
	kDefaultChannelBufferSize  = 50
	kDefaultMaximumMessageSize = 4 * 1024 * 1024

	LatencyNotSpecified time.Duration = 1 * time.Millisecond // unspecified latency, default to 1ms
	LatencyZero         time.Duration = 0                    // never updates latency

	AppName = "actor-platform"
)

var (
	RequestTimeout     = kDefaultSystemTimeout    // system requests
	FlowTimeout        = kDefaultSystemTimeout    // flow object requests
	ExecutionTimeout   = kDefaultExecutionTimeout // actor execution
	ChannelBufferSize  = kDefaultChannelBufferSize
	MaximumMessageSize = kDefaultMaximumMessageSize
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
