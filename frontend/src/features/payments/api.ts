import { apiClient } from '../../lib/api-client';
import type { Payment } from '../../types/api';

export const paymentsApi = {
  get: (paymentId: string) =>
    apiClient.get<Payment>(`/api/v1/payments/${paymentId}`),

  listByOrder: (orderId: string) =>
    apiClient.get<{ payments: Payment[] }>(`/api/v1/orders/${orderId}/payments`),

  refund: (paymentId: string, reason: string) =>
    apiClient.post<{ paymentId: string; status: string }>(`/api/v1/payments/${paymentId}/refund`, { reason }),
};
