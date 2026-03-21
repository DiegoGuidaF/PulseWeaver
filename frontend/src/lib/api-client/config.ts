import { client } from '../api/client.gen';

// Configure the client once at app startup
client.setConfig({
  baseUrl: '/api/v1',
  credentials: 'include', // Send cookies with requests
});

if (import.meta.env.DEV) {
  client.interceptors.error.use((error, _response, request) => {
    console.error('[api]', request.method, request.url, error);
    return error;
  });
}

export { client };
