// ---- Auth ----
export interface RegisterRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
}

export interface RegisterResponse {
  userId: string;
  email: string;
  authToken: string;
  createdAt: string;
}

export interface LoginResponse {
  accessToken: string;
  userId: string;
  expiresAt: string;
}

export interface User {
  userId: string;
  email: string;
  firstName: string;
  lastName: string;
  createdAt: string;
  updatedAt: string;
}

// ---- Products ----
export interface Product {
  productId: string;
  name: string;
  description: string;
  priceCents: string;
  currency: string;
  stockQuantity: number;
  sku: string;
  createdAt: string;
  updatedAt: string;
}

export interface ProductListResponse {
  products: Product[];
  pagination: {
    totalCount: number;
    page: number;
    pageSize: number;
  };
}

export interface CreateProductRequest {
  name: string;
  description: string;
  price_cents: number;
  currency: string;
  stock_quantity: number;
  sku: string;
}

// ---- Orders ----
export interface OrderItem {
  productId: string;
  productName: string;
  quantity: number;
  priceCents: string;
}

export interface Order {
  orderId: string;
  userId: string;
  items: OrderItem[];
  totalCents: string;
  currency: string;
  status: string;
  paymentId: string;
  createdAt: string;
  updatedAt: string;
}

export interface PlaceOrderRequest {
  user_id: string;
  items: { product_id: string; quantity: number }[];
  currency: string;
}

export interface PlaceOrderResponse {
  orderId: string;
  status: string;
  totalCents: string;
  createdAt: string;
}

// ---- Payments ----
export interface Payment {
  paymentId: string;
  orderId: string;
  amountCents: string;
  currency: string;
  status: string;
  method: string;
  createdAt: string;
  updatedAt: string;
}

// ---- Common ----
export interface ApiError {
  code: number;
  message: string;
  details: unknown[];
}
