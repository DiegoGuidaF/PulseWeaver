import {defineConfig} from '@hey-api/openapi-ts';

export default defineConfig({
    input: '../api/openapi.yaml',
    output: {
        path: './src/lib/api',
        postProcess: [
            'prettier',
            'eslint'
        ],
    },
    plugins: [
        '@hey-api/schemas',
        {
            dates: true,
            bigInt: false,
            name: '@hey-api/transformers',
        },
        {
            enums: 'javascript',
            name: '@hey-api/typescript',
        },
        {
            name: '@hey-api/sdk',
            transformer: true,
            validator: true
        },
        '@tanstack/react-query',
        '@hey-api/client-fetch',
        {
            name: 'zod',
            requests: true,
            responses: true,
            definitions: true,
            dates: {
                offset: true,  // Allow values with timezone offsets
                local: true,   // Allow values without timezone
            },
        }
    ],
});
