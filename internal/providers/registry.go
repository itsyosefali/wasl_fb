package providers

import (
	"fmt"
	"sync"
)

// Registry holds all channel providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[Channel]Provider
}

func NewRegistry(items ...Provider) *Registry {
	r := &Registry{providers: make(map[Channel]Provider)}
	for _, p := range items {
		r.Register(p)
	}
	return r
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Channel()] = p
}

func (r *Registry) Get(ch Channel) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[ch]
	if !ok {
		return nil, fmt.Errorf("provider not registered: %s", ch)
	}
	return p, nil
}

func (r *Registry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}

// ResolveChannel maps platform string to Channel.
func ResolveChannel(platform string) Channel {
	switch platform {
	case string(ChannelInstagram):
		return ChannelInstagram
	case string(ChannelWhatsApp):
		return ChannelWhatsApp
	case string(ChannelTelegram):
		return ChannelTelegram
	default:
		return ChannelFacebook
	}
}
