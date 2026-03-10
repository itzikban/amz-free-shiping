# Test Cases

## Case A: Same product, two destinations
- URL: `https://www.amazon.com/dp/B0DHCZBKW7`
- Destination 1: `US`, zip `10013`
- Destination 2: `IL`

Expected:
- US can resolve `free_shipping_country=true`
- IL should remain `free_shipping_country=false` unless explicit Israel signal appears

## Case B: Captcha/blocked page
If upstream page includes captcha markers:
- `signal` should include `captcha_detected`
- `free_shipping_country` should be `false`

## Case C: Generic free shipping only
If response contains generic free shipping text but no destination confirmation:
- `free_shipping_country=false`
- no country-specific alert
