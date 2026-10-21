package main

import (
	"context"
	"errors"
	"testing"
)

type recordingPublisher struct {
	calls int
	body  any
}

func (p *recordingPublisher) post(_ context.Context, _, _ string, body any) error {
	p.calls++
	p.body = body
	return nil
}

func TestMarketplaceDecision(t *testing.T) {
	for _, tc := range []struct {
		name    string
		budget  int
		buyer   string
		allowed bool
	}{
		{"below ask", 49, "buyer-1", false},
		{"meets ask", 50, "buyer-1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pub := &recordingPublisher{}
			m := newMarket(pub)
			ctx := context.Background()
			if _, err := m.register(ctx, asset{ID: "camera", Seller: "seller-1", Ask: 50}, "create-1"); err != nil {
				t.Fatal(err)
			}
			got, err := m.match(ctx, "camera", tc.buyer, tc.budget, "match-1")
			if (err == nil) != tc.allowed {
				t.Fatalf("match=%+v err=%v", got, err)
			}
			if !tc.allowed {
				if pub.calls != 1 {
					t.Fatalf("rejected match published: %d calls", pub.calls)
				}
				return
			}
			if got.State != "reserved" || pub.calls != 2 {
				t.Fatalf("match=%+v calls=%d", got, pub.calls)
			}
			if body := pub.body.(map[string]any); body["account_id"] != nil {
				t.Fatalf("domain seller sent as account_id: %+v", body)
			}
			if _, err := m.handoff(ctx, "camera", "other-buyer", "order-1", "wrong-1"); err == nil {
				t.Fatal("different buyer handed off order")
			}
			order, err := m.handoff(ctx, "camera", tc.buyer, "order-1", "handoff-1")
			if err != nil || order.State != "handed_off" || order.Order != "order-1" {
				t.Fatalf("handoff=%+v err=%v", order, err)
			}
			if _, err := m.handoff(ctx, "camera", tc.buyer, "order-1", "handoff-1"); err != nil || pub.calls != 3 {
				t.Fatalf("retry err=%v calls=%d", err, pub.calls)
			}
		})
	}
}

func TestNoPublishOnFailedCreate(t *testing.T) {
	m := newMarket(failingPublisher{})
	if _, err := m.register(context.Background(), asset{ID: "camera", Seller: "seller", Ask: 1}, "create-1"); err == nil {
		t.Fatal("expected create error")
	}
	if len(m.assets) != 0 {
		t.Fatal("asset registered without channel")
	}
}

type failingPublisher struct{}

func (failingPublisher) post(context.Context, string, string, any) error {
	return errors.New("request failed")
}
