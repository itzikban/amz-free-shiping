# amz-free-shiping (Monorepo)

![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/itzikban/amz-free-shiping?utm_source=oss&utm_medium=github&utm_campaign=itzikban%2Famz-free-shiping&labelColor=171717&color=FF570A&link=https%3A%2F%2Fcoderabbit.ai&label=CodeRabbit+Reviews)

This repository is structured as a **monorepo** for multiple services.

## Current layout

- `backend/` — Go backend service (free-shipping checker API)
- `frontend/` — Next.js responsive web app

## Planned expansion

- `backend/` — API + workers + schedulers
- `frontend/` — dashboard + user flows
- additional services as needed (`services/*`)

## Backend quick start

```bash
cd backend
cp .env.example .env
# set DECODO_BASIC_AUTH and GOSCRAPE_BIND_IP in .env (see WireGuard section below)

go mod tidy
go run ./cmd/server
```

API runs on `:8085` by default.

## WireGuard (Mullvad) setup

The goscrape fallback routes Amazon search traffic through a Mullvad WireGuard tunnel to avoid datacenter IP blocks. The tunnel uses `Table = off` so **only goscrape traffic is affected — SSH and all other server traffic is untouched.**

### First-time setup

Requirements: Mullvad account, Ubuntu/Debian server with root access.

```bash
# 1. Install WireGuard
sudo apt-get install -y wireguard wireguard-tools

# 2. Generate a key pair
wg genkey | tee /tmp/wg-private.key | wg pubkey > /tmp/wg-public.key

# 3. Register the public key with Mullvad and get your tunnel IP
ACCESS_TOKEN=$(curl -s -X POST https://api.mullvad.net/auth/v1/token \
  -H "Content-Type: application/json" \
  -d '{"account_number": "YOUR_MULLVAD_ACCOUNT"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

curl -s -X POST https://api.mullvad.net/app/v1/wireguard-keys \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"pubkey\": \"$(cat /tmp/wg-public.key)\"}"
# Note the ipv4_address from the response (e.g. 10.x.x.x)

# 4. Write the config (replace values with your own)
sudo tee /etc/wireguard/mullvad-nyc.conf > /dev/null <<EOF
[Interface]
PrivateKey = $(cat /tmp/wg-private.key)
Address = 10.x.x.x/32
Table = off
PostUp = ip route add default dev mullvad-nyc table 200; ip rule add from 10.x.x.x lookup 200 priority 100
PreDown = ip rule del from 10.x.x.x lookup 200 priority 100 2>/dev/null; ip route del default dev mullvad-nyc table 200 2>/dev/null; true

[Peer]
PublicKey = IzqkjVCdJYC1AShILfzebchTlKCqVCt/SMEXolaS3Uc=
Endpoint = 143.244.47.65:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
EOF

# 5. Clean up temporary key files (must run AFTER config is written above)
shred -u /tmp/wg-private.key /tmp/wg-public.key 2>/dev/null || rm -f /tmp/wg-private.key /tmp/wg-public.key

# 6. Bring up the tunnel
sudo wg-quick up mullvad-nyc

# 7. Enable auto-start on boot
sudo systemctl enable wg-quick@mullvad-nyc

# 8. Set the WireGuard IP in backend/.env
echo "GOSCRAPE_BIND_IP=10.x.x.x" >> backend/.env
```

### Check tunnel status

```bash
sudo wg show mullvad-nyc          # shows handshake + transfer stats
ip rule show | grep 10.x.x.x     # confirms policy rule is active
```

### If the tunnel goes down

```bash
sudo wg-quick up mullvad-nyc
```

If it was already up (interface exists error):

```bash
sudo wg-quick down mullvad-nyc && sudo wg-quick up mullvad-nyc
```

### Smoke test goscrape through the tunnel

```bash
cd backend
GOSCRAPE_BIND_IP=10.x.x.x go run ./cmd/goscrape-test/ "wireless earbuds"
```

Should return ≥1 result with valid ASINs and prices. If it returns 0 results, check `sudo wg show` for a recent handshake timestamp.

## Implemented features

### Backend
- Country-aware `/check` endpoint (`US` with ZIP, `IL` strict destination logic)
- Decodo integration as primary scraper source
- Captcha/blocked-page detection signaling
- Initial PostgreSQL schema migration (`backend/migrations/001_init.sql`)
- Redis queue runtime with retry + dead-letter flow
- Scheduler service for enqueueing due checks

### Frontend
- Responsive Next.js app in `frontend/`
- URL + country + ZIP check form
- Backend health indicator (live polling)
- Strict result display (`free_shipping` vs `free_shipping_country`)
- Raw backend response viewer for debugging

## Notes
- Feature implementation docs are tracked under `docs/features/`.
- End-to-end smoke check script: `tests/smoke-ui-backend.sh`


## Where to start (planning + execution)

1. Read product plan summary: `docs/PRODUCT_PLAN.md`
2. Read full original planning request: `docs/PRODUCT_PLAN_REQUEST.md`
3. Open static UI perspective mock: `mock-ui/index.html`
4. Pick next Linear ticket from team `ITZ` backlog
5. Implement in small PRs with tests + verification notes

## Ticket tracking rule (important)
When a ticket is closed, update **both**:
- Linear issue state -> `Done`
- matching local feature doc under `docs/features/ITZ-XX.md` with `Status: Closed ✅`

This keeps repo docs and Linear in sync and makes progress easy to track across sessions.

## Working process
- See `WORKING-AGREEMENT.md` for collaboration rules, delivery workflow, and definition of done.
