// Re-export generated API types and functions
export * from '../api';

// Re-export client config and error utilities
export { client } from './config';
export { ApiError, toApiError, toErrorMessage } from './errors';
