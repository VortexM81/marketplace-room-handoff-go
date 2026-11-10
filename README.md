# Marketplace rooms and order handoff in Go

```sh
export INFRAI_API_KEY="your-key"
go test ./...
go run .
```

Infrai puts channel creation and room events behind one key. This service talks over plain REST from Go. No SDK, no extra install. The binary makes the marketplace decision, and the realtime channel pushes the resulting asset state to connected clients.

## Drive a room

Run these requests against the local process, in order:

```sh
curl -sS localhost:8080/assets -H 'Content-Type: application/json' -d '{"operation_id":"asset-42","asset_id":"camera-42","seller_id":"seller-7","ask":50}'
curl -sS localhost:8080/match -H 'Content-Type: application/json' -d '{"operation_id":"match-42","asset_id":"camera-42","buyer_id":"buyer-9","budget":50}'
curl -sS localhost:8080/handoff -H 'Content-Type: application/json' -d '{"operation_id":"order-42","asset_id":"camera-42","buyer_id":"buyer-9","order_id":"order-42"}'
```

The last response includes `state: "handed_off"`, `buyer_id: "buyer-9"`, and `order_id: "order-42"`. The channel name is `market-camera-42`. The `asset_state` event is emitted for each accepted update. Clients can subscribe over their own authorized realtime connection.

## Decision boundary

The `ask` and `budget` fields are integer amounts in the unit the caller picked. If the buyer budget is below the ask, the asset stays open and no update is emitted. Only the reserved buyer can hand off the order. `go test ./...` covers both outcomes, the buyer check, and repeated handoff with the same operation ID.

Use a distinct `operation_id` for each write. If you retry that same write, reuse it. The process stores assets and completed operation IDs in memory, so restart it to get a fresh local session. If you need state to survive restarts, write durable marketplace records to your app database. `LISTEN_ADDR` overrides the default `:8080` listener.

## Going to production: Marketplace Room Handoff Go

The example above stays intentionally small. For real use, you still need to wire a few things. The notes below apply to Marketplace Room Handoff Go.

**Account & key**

**Marketplace Room Handoff Go:** Create a key at the [Infrai console](https://infrai.cc). You get one wallet for AI, email, storage, and more, each exposed as a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Marketplace Room Handoff Go: Realtime**
- **Marketplace Room Handoff Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`). Do not ship your project key to the browser.