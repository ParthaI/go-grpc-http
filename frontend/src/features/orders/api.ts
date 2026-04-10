import { apiClient } from '../../lib/api-client';
import type { Order, PlaceOrderRequest, PlaceOrderResponse } from '../../types/api';

export const ordersApi = {
  get: (orderId: string) =>
    apiClient.get<Order>(`/api/v1/orders/${orderId}`),

  listByUser: (userId: string) =>
    apiClient.get<{ orders: Order[] }>(`/api/v1/users/${userId}/orders`),

  place: (data: PlaceOrderRequest) =>
    apiClient.post<PlaceOrderResponse>('/api/v1/orders', data),

  cancel: (orderId: string, reason: string) =>
    apiClient.post<{ orderId: string; status: string }>(`/api/v1/orders/${orderId}/cancel`, { reason }),
};
