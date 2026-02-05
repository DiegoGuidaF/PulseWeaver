import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query' // Import these
import App from './App'
import './index.css'

// 1. Create the client
const queryClient = new QueryClient()

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        {/* 2. Wrap the App */}
        <QueryClientProvider client={queryClient}>
            <App />
        </QueryClientProvider>
    </React.StrictMode>,
)
