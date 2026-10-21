package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type publisher interface {
	post(context.Context, string, string, any) error
}

type asset struct {
	ID     string `json:"asset_id"`
	Seller string `json:"seller_id"`
	Ask    int    `json:"ask"`
	Buyer  string `json:"buyer_id,omitempty"`
	Order  string `json:"order_id,omitempty"`
	State  string `json:"state"`
}

type market struct {
	mu         sync.Mutex
	api        publisher
	assets     map[string]asset
	operations map[string]asset
}

func newMarket(api publisher) *market {
	return &market{api: api, assets: make(map[string]asset), operations: make(map[string]asset)}
}

func (m *market) register(ctx context.Context, a asset, operation string) (asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if previous, ok := m.operations[operation]; ok {
		return previous, nil
	}
	if operation == "" || a.ID == "" || a.Seller == "" || a.Ask <= 0 {
		return asset{}, errors.New("operation_id, asset_id, seller_id and positive ask required")
	}
	if _, exists := m.assets[a.ID]; exists {
		return asset{}, errors.New("asset already registered")
	}
	channel := "market-" + a.ID
	if err := m.api.post(ctx, "/v1/realtime/channel/create", operation+":channel", map[string]any{"channel": channel, "type": "public"}); err != nil {
		return asset{}, err
	}
	a.State = "open"
	m.assets[a.ID] = a
	m.operations[operation] = a
	return a, nil
}

func (m *market) match(ctx context.Context, id, buyer string, budget int, operation string) (asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if previous, ok := m.operations[operation]; ok {
		return previous, nil
	}
	if operation == "" || buyer == "" || budget <= 0 {
		return asset{}, errors.New("operation_id, buyer_id and positive budget required")
	}
	a, ok := m.assets[id]
	if !ok {
		return asset{}, errors.New("asset not found")
	}
	if a.State != "open" {
		return asset{}, errors.New("asset is reserved")
	}
	if budget < a.Ask {
		return asset{}, errors.New("budget below ask")
	}
	a.Buyer, a.State = buyer, "reserved"
	if err := m.publish(ctx, a, operation); err != nil {
		return asset{}, err
	}
	m.assets[id], m.operations[operation] = a, a
	return a, nil
}

func (m *market) handoff(ctx context.Context, id, buyer, order, operation string) (asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if previous, ok := m.operations[operation]; ok {
		return previous, nil
	}
	if operation == "" || buyer == "" || order == "" {
		return asset{}, errors.New("operation_id, buyer_id and order_id required")
	}
	a, ok := m.assets[id]
	if !ok {
		return asset{}, errors.New("asset not found")
	}
	if a.State != "reserved" || a.Buyer != buyer {
		return asset{}, errors.New("order requires the reserved buyer")
	}
	a.Order, a.State = order, "handed_off"
	if err := m.publish(ctx, a, operation); err != nil {
		return asset{}, err
	}
	m.assets[id], m.operations[operation] = a, a
	return a, nil
}

func (m *market) publish(ctx context.Context, a asset, operation string) error {
	return m.api.post(ctx, "/v1/realtime/publish", operation+":event", map[string]any{
		"channel": fmt.Sprintf("market-%s", a.ID), "event": "asset_state", "data": a,
	})
}
