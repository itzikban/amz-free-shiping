# Frontend Testing Guide

## Manual checklist

1. Open app on desktop and mobile widths.
2. Enter sample URL: `https://www.amazon.com/dp/B0DHCZBKW7`
3. Test US context:
   - Country: `US`
   - ZIP: `10013`
   - Click `Check free shipping`
   - Expect destination verdict based on backend response.
4. Test IL context:
   - Country: `IL`
   - ZIP disabled
   - Click check
   - Expect IL-specific result.
5. Verify loading state and error state when backend is down.

## Backend dependency
Frontend does not compute shipping itself; it reflects backend truth from `/check`.

## Expected behavior
- Button disabled when URL invalid.
- Button disabled for US when ZIP empty.
- Result panel shows method/signal/checked timestamp.
