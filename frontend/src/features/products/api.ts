import { apiClient } from '../../lib/api-client';
import type { Product, ProductListResponse, CreateProductRequest } from '../../types/api';

export const productsApi = {
  list: () =>
    apiClient.get<ProductListResponse>('/api/v1/products'),

  get: (productId: string) =>
    apiClient.get<Product>(`/api/v1/products/${productId}`),

  create: (data: CreateProductRequest) =>
    apiClient.post<Product>('/api/v1/products', data),

  update: (productId: string, data: { name: string; description: string; price_cents: number; currency: string }) =>
    apiClient.put<Product>(`/api/v1/products/${productId}`, data),

  updateInventory: (productId: string, quantity: number) =>
    apiClient.put<Product>(`/api/v1/products/${productId}/inventory`, { quantity }),
};
