import { client } from '../api/client.gen';

// Configure the client once at app startup
client.setConfig({
  baseUrl: '/api/v1',
  credentials: 'include', // Send cookies with requests
});

export { client };
