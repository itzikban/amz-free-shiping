#!/usr/bin/env bash
# find_alternatives.sh
# Fetches 3 alternatives with free shipping for an Amazon product.
# Strategy: amazon_product + markdown:true → parse "Similar items" section.
#
# Usage:   ./find_alternatives.sh [ASIN] [COUNTRY]
# Example: ./find_alternatives.sh B0DHCZBKW7 NL

set -euo pipefail

ASIN="${1:-B0DHCZBKW7}"
COUNTRY="${2:-NL}"
AUTH="VTAwMDAzNjc2MTU6UFdfMTdiOWVmN2M0OGM4ZjRkZDk0YWExMzU2MDk4NjdmNmMy"
DECODO_URL="https://scraper-api.decodo.com/v2/scrape"
TMPFILE="/tmp/decodo_product_${ASIN}.json"

log() { echo "▶ $*" >&2; }
die() { echo "✗ $*" >&2; exit 1; }

command -v curl    &>/dev/null || die "curl not found"
command -v python3 &>/dev/null || die "python3 not found"

# ── Step 1: fetch product page as markdown ────────────────────────────────────
log "Fetching product page for ASIN=$ASIN (markdown mode)…"

curl -sf --request POST "$DECODO_URL" \
  --header "Accept: application/json" \
  --header "Authorization: Basic $AUTH" \
  --header "Content-Type: application/json" \
  --data "{
    \"target\": \"amazon_product\",
    \"query\": \"$ASIN\",
    \"headless\": \"html\",
    \"autoselect_variant\": true,
    \"markdown\": true
  }" -o "$TMPFILE"

log "Response saved to $TMPFILE ($(wc -c < "$TMPFILE") bytes)"

# ── Step 2: parse alternatives from markdown ──────────────────────────────────
log "Extracting alternatives…"

# Pass bash vars as env so the heredoc can be fully single-quoted (no bash expansion)
ASIN="$ASIN" COUNTRY="$COUNTRY" TMPFILE="$TMPFILE" python3 << 'PYEOF'
import json, re, sys, os

asin    = os.environ['ASIN']
country = os.environ['COUNTRY']
tmpfile = os.environ['TMPFILE']

with open(tmpfile) as f:
    d = json.load(f)

r       = d['results'][0]
status  = r.get('status_code', 0)
content = r.get('content', '')

if not isinstance(content, str) or len(content) < 1000:
    print(f"ERROR: unexpected response (status={status}, len={len(str(content))})", file=sys.stderr)
    sys.exit(1)

# Product title from first # heading
title_m = re.search(r'^# (.{20,})', content, re.MULTILINE)
product_title = title_m.group(1).strip() if title_m else 'Unknown'
print(f"Product: {product_title[:90]}")
print()

# ── Find "Similar items that may deliver to you quickly" section ──────────────
section_start = content.find('## Similar items that may deliver to you quickly')
if section_start < 0:
    section_start = content.find('## More items to explore')
if section_start < 0:
    print("ERROR: no similar-items section found in markdown", file=sys.stderr)
    sys.exit(1)

end_idx = len(content)
for marker in ['## Product information', '## Frequently bought together', '## Product description']:
    pos = content.find(marker, section_start + 50)
    if 0 < pos < end_idx:
        end_idx = pos

section = content[section_start:end_idx]
items   = re.split(r'\n\d+\. ', section)[1:]  # drop section header

results = []
for item in items:
    if len(results) >= 3:
        break

    # ASIN
    asin_m = re.search(r'/dp/([A-Z0-9]{10})', item)
    if not asin_m:
        continue
    item_asin = asin_m.group(1)

    # Image URL
    img_m   = re.search(r'!\[[^\]]*\]\((https?://[^\)]+)\)', item)
    img_url = img_m.group(1) if img_m else ''

    # Title: first standalone [text](url) that doesn't start with !
    title_m = re.search(r'(?<!!)\[([^\]]{20,})\]\((?:/[A-Za-z]|https?://www\.amazon)', item)
    title = title_m.group(1).strip() if title_m else ''
    if not title:
        alt_m = re.search(r'!\[([^\]]{20,})\]', item)
        title = alt_m.group(1).strip() if alt_m else ''

    # Price — markdown duplicates it: [$59.49$59.49](url)
    # The pattern \[\$N.NN captures the first occurrence
    price_m = re.search(r'\[\$([0-9,]+\.[0-9]{2})', item)
    if not price_m:
        price_m = re.search(r'\$([0-9,]+\.[0-9]{2})', item)
    price = float(price_m.group(1).replace(',', '')) if price_m else 0.0

    free_ship = 'FREE Shipping' in item or 'free shipping' in item.lower()

    if item_asin and title:
        results.append({
            'asin':         item_asin,
            'title':        title,
            'price':        price,
            'image':        img_url,
            'free_shipping': free_ship,
            'url':          f'https://www.amazon.com/dp/{item_asin}'
        })

if not results:
    print("ERROR: no alternatives parsed — check raw file", file=sys.stderr)
    sys.exit(1)

# ── Human-readable output ─────────────────────────────────────────────────────
print('=' * 60)
print(f'  3 ALTERNATIVES WITH FREE SHIPPING  (country hint: {country})')
print('=' * 60)
for i, alt in enumerate(results, 1):
    free_label = 'YES' if alt['free_shipping'] else 'NO'
    print(f"""
[{i}] {alt['title'][:90]}
    ASIN  : {alt['asin']}
    URL   : {alt['url']}
    Price : ${alt['price']}
    Free? : {free_label}
    Image : {alt['image'][:80]}""")

print()
print('--- JSON ---')
print(json.dumps(results, indent=2))
PYEOF
