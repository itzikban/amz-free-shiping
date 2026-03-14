# HTTPS Setup for wizflo.shop

## Prerequisites (already done)

- [x] DNS A records: `@` and `www` → `20.240.201.161`
- [x] Nginx installed and configured on server
- [x] Certbot + python3-certbot-nginx installed
- [x] Nginx config at `/etc/nginx/sites-enabled/wizflo.shop`

## Once DNS propagates, run:

```bash
ssh ubuntu@20.240.201.161
```

### 1. Verify DNS is working

```bash
host wizflo.shop 8.8.8.8
# Should return: wizflo.shop has address 20.240.201.161
```

### 2. Get SSL certificate

```bash
sudo certbot --nginx -d wizflo.shop -d www.wizflo.shop --non-interactive --agree-tos -m itzikban@gmail.com
```

### 3. Verify HTTPS

```bash
curl -I https://wizflo.shop
# Should return 200 OK
```

### 4. Verify auto-renewal

```bash
sudo certbot renew --dry-run
```

Certbot auto-renews via systemd timer. Check with:

```bash
sudo systemctl list-timers | grep certbot
```

## What certbot does automatically

- Modifies `/etc/nginx/sites-enabled/wizflo.shop` to add SSL config
- Adds redirect from HTTP (80) → HTTPS (443)
- Sets up auto-renewal (certs expire every 90 days)

## Troubleshooting

If certbot fails:

```bash
# Check nginx is running
sudo systemctl status nginx

# Check port 80 is open (needed for ACME challenge)
sudo ufw status
# If firewall blocks 80/443:
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Check nginx config
sudo nginx -t

# Check certbot logs
sudo cat /var/log/letsencrypt/letsencrypt.log | tail -50
```
