package utils

import (
	"github.com/asynkron/protoactor-go/actor"
)

func IsSameSystem(pids ...*actor.PID) bool {
	if len(pids) == 0 {
		return true
	}

	address := pids[0].Address
	for _, pid := range pids {
		if pid == nil {
			continue
		}
		if pid.Address != address {
			return false
		}
	}

	return true
}
