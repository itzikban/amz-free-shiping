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

export type FillSuggestion = {
  asin: string;
  title: string;
  price_usd: number;
  image_url?: string;
  url: string;
  free_shipping_hint: boolean;
  score: number;
};

export type FillCombo = {
  items: FillSuggestion[];
  total: number;
  score: number;
};

export type FillToThresholdResponse = {
  current_price: number;
  missing_amount: number;
  suggestions: FillSuggestion[];
  combos: FillCombo[];
};
