import { apiClient } from '../../lib/api-client';
import type { RegisterRequest, RegisterResponse, LoginResponse, User } from '../../types/api';

export const authApi = {
  register: (data: RegisterRequest) =>
    apiClient.post<RegisterResponse>('/api/v1/users/register', data),

  login: (email: string, password: string) =>
    apiClient.post<LoginResponse>('/api/v1/users/login', { email, password }),

  getUser: (userId: string) =>
    apiClient.get<User>(`/api/v1/users/${userId}`),
};
