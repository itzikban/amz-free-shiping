export type TrackedProduct = {
  id: string;
  url: string;
  country: string;
  zip?: string;
  created_at?: string;
  last_checked_at: string;
  last_price_usd?: number;
  free_shipping_country: boolean;
  free_shipping?: boolean;
  signal: string;
  method?: string;
};

export type UserAlert = {
  id: string;
  message: string;
  created_at: string;
};

export const FALLBACK_TRACKED_PRODUCTS: TrackedProduct[] = [
  {
    id: "mock-item-1",
    url: "https://www.amazon.com/dp/B0DHCZBKW7",
    country: "US",
    zip: "10013",
    last_checked_at: new Date().toISOString(),
    last_price_usd: 29.99,
    free_shipping_country: true,
    signal: "mock_fallback",
    method: "mock",
  },
  {
    id: "mock-item-2",
    url: "https://www.amazon.com/dp/B09G3HRMVB",
    country: "IL",
    last_checked_at: new Date(Date.now() - 1000 * 60 * 22).toISOString(),
    last_price_usd: 54.4,
    free_shipping_country: false,
    signal: "mock_fallback",
    method: "mock",
  },
];

export const FALLBACK_ALERTS: UserAlert[] = [
  {
    id: "mock-alert-1",
    message: "✅ Free shipping became available for one tracked product.",
    created_at: new Date(Date.now() - 1000 * 60 * 3).toISOString(),
  },
  {
    id: "mock-alert-2",
    message: "📉 Price changed on a tracked product.",
    created_at: new Date(Date.now() - 1000 * 60 * 21).toISOString(),
  },
];
