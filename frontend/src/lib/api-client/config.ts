import { client } from '../api/client.gen';

// Configure the client once at app startup
client.setConfig({
  baseUrl: '/api/v1',
  credentials: 'include', // Send cookies with requests
});

// Attach the HTTP status code to the error object so toApiError() can read it.
client.interceptors.error.use((error, response) => {
  if (response && error && typeof error === 'object') {
    (error as Record<string, unknown>).status = response.status;
  }
  return error;
});

if (import.meta.env.DEV) {
  client.interceptors.error.use((error, _response, request) => {
    console.error('[api]', request.method, request.url, error);
    return error;
  });
}

export { client };
