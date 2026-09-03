package engine

import "sync"

type Manager struct {
	engines map[string]*Engine
	mu      sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		engines: make(map[string]*Engine),
	}
}

func (m *Manager) GetEngine(symbol string) *Engine {
	m.mu.RLock()
	engine, exsit := m.engines[symbol]
	m.mu.RUnlock()

	if exsit {
		return engine
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.engines[symbol] = NewEngine(symbol)
	return m.engines[symbol]

}
