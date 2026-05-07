package runtimewatcher

import (
	"github.com/go-logr/logr"
	watchertypes "github.com/kyma-project/runtime-watcher/listener/pkg/v2/types"
)

type RegistryCacheConfigListener struct {
	Addr   string
	Logger logr.Logger
	events chan watchertypes.GenericEvent
}

func NewRegistryCacheConfigListener(addr string) *RegistryCacheConfigListener {
	return &RegistryCacheConfigListener{
		Addr: addr,
	}
}
