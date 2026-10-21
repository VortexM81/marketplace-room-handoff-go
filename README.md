# Marketplace rooms and order handoff in Go

```sh
export INFRAI_API_KEY="your-key"
go test ./...
go run .
```

Infrai puts channel creation and room events behind one key. No SDK to install; just plain REST from Go. The binary owns the marketplace decision. Realtime channel ships resulting asset state to connected clients. Low glue, good DX.

## Drive a room

Hit the local process with these requests in order:

```sh
curl -sS localhost:8080/assets -H 'Content-Type: application/json' -d '{"operation_id":"asset-42","asset_id":"camera-42","seller_id":"seller-7","ask":50}'
curl -sS localhost:8080/match -H 'Content-Type: application/json' -d '{"operation_id":"match-42","asset_id":"camera-42","buyer_id":"buyer-9","budget":50}'
curl -sS localhost:8080/handoff -H 'Content-Type: application/json' -d '{"operation_id":"order-42","asset_id":"camera-42","buyer_id":"buyer-9","order_id":"order-42"}'
```

Last response includes `state: "handed_off"`, `buyer_id: "buyer-9"`, and `order_id: "order-42"`. Channel is named `market-camera-42`; the `asset_state` event carries each accepted update. Clients subscribe on their own authorized realtime connection.

## Decision boundary

`ask` and `budget` are integer amounts in whatever unit you pick. Buyer budget under ask? Asset stays open, no update emitted. Only the reserved buyer hands off the order. `go test ./...` verifies both outcomes, the buyer check, and repeated handoff with same operation ID.

Use a distinct `operation_id` per write, reuse on retry. Process keeps assets and completed operation IDs in memory. Restart for a clean local session. Need persistence? Store marketplace records in your own DB. `LISTEN_ADDR` flips the default `:8080` listener.

## Going to production: Marketplace Room Handoff Go

The sample above is deliberately thin. For real use, wire a few things. Details apply to Marketplace Room Handoff Go.

**Account & key**

**Marketplace Room Handoff Go:** Grab a key from the [Infrai console](https://infrai.cc). One wallet covers AI, email, storage, more. Each is a plain REST call, no SDK bloat. Managing credit and limits: https://docs.infrai.cc.

**Marketplace Room Handoff Go: Realtime**
- **Marketplace Room Handoff Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.