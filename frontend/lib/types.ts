export type Alternative = {
  asin: string;
  title: string;
  url: string;
  image_url?: string;
  price_usd?: number;
  free_shipping: boolean;
};

export type CheckResponse = {
  url: string;
  country: string;
  checked_at: string;
  price_usd?: number;
  free_shipping: boolean;
  free_shipping_country: boolean;
  signal: string;
  title?: string;
  image_url?: string;
  alternatives?: Alternative[];
};
