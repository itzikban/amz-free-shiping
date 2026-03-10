export type CheckResponse = {
  url: string;
  country: string;
  checked_at: string;
  price_usd?: number;
  free_shipping: boolean;
  free_shipping_country: boolean;
  signal: string;
  method: string;
};
