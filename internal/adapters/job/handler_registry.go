package job

import (
	"sync"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type HandlerRegistry struct {
	mutex    sync.RWMutex
	handlers map[string]ports.JobHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[string]ports.JobHandler)}
}

func (registry *HandlerRegistry) Register(jobType string, handler ports.JobHandler) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	registry.handlers[jobType] = handler
}

func (registry *HandlerRegistry) Get(jobType string) (ports.JobHandler, bool) {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	handler, ok := registry.handlers[jobType]
	return handler, ok
}
