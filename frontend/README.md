# Frontend (Next.js)

Modern responsive UI for checking free-shipping status against the Go backend.

## Features
- Product URL input
- Destination selector (`US` / `IL`)
- ZIP input for US
- Calls backend check API through a Next.js route handler
- Clear result card: free shipping for destination or not
- Mobile-friendly responsive layout

## Run locally
```bash
cd frontend
npm install
npm run dev
```

Open: `http://localhost:3000`

## Environment
Set backend URL for proxy route:

```bash
BACKEND_BASE_URL=http://127.0.0.1:8085
```

You can place it in `frontend/.env.local`.

## API flow
UI -> `frontend/app/api/check/route.ts` -> backend `/check`

This avoids CORS issues and keeps backend URL configurable.

## Production notes
- Build with `npm run build`
- Start with `npm run start`
- Put behind reverse proxy (Nginx/Caddy)
- Keep `BACKEND_BASE_URL` private server-side env
