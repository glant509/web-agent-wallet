package mcp

import "sync"

type Server struct {
	Name        string
	Description string
	Endpoint    string
}

type Registry struct {
	mu      sync.RWMutex
	servers map[string]Server
}

func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]Server),
	}
}

func (r *Registry) Register(server Server) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers[server.Name] = server
}

func (r *Registry) List() []Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]Server, 0, len(r.servers))
	for _, server := range r.servers {
		servers = append(servers, server)
	}
	return servers
}
