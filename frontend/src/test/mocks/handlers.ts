import { http, HttpResponse } from 'msw';
import { createMockDevice } from './data';
import { createMockUser } from './data';

export const handlers = [
  // GET /api/v1/devices - Returns list of devices
  http.get('/api/v1/devices', () => {
    return HttpResponse.json([createMockDevice()]);
  }),

  // GET /api/v1/auth/me - Returns current user
  http.get('/api/v1/auth/me', () => {
    return HttpResponse.json(createMockUser());
  }),
];
