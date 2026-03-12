/**
 * Client-facing tracked product model used by the watchlist UI.
 */
export type TrackedProduct = {
  id: string;
  url: string;
  country: string;
  zip?: string;
  created_at?: string;
  last_checked_at: string;
  last_price_usd?: number;
  /**
   * Required destination-policy flag: whether the offer supports free shipping
   * for the selected country/ZIP context.
   */
  free_shipping_country: boolean;
  /**
   * Optional listing/runtime flag for a specific offer snapshot.
   * May be undefined when this signal is unavailable.
   */
  free_shipping?: boolean;
  signal: string;
  method?: string;
};

export type UserAlert = {
  id: string;
  message: string;
  created_at: string;
  type?: "free_shipping" | "price_change" | "other";
};

export function getFallbackTrackedProducts(): TrackedProduct[] {
  return [
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
}

export function getFallbackAlerts(): UserAlert[] {
  return [
    {
      id: "mock-alert-1",
      message: "✅ Free shipping became available for one tracked product.",
      created_at: new Date(Date.now() - 1000 * 60 * 3).toISOString(),
      type: "free_shipping",
    },
    {
      id: "mock-alert-2",
      message: "📉 Price changed on a tracked product.",
      created_at: new Date(Date.now() - 1000 * 60 * 21).toISOString(),
      type: "price_change",
    },
  ];
}
