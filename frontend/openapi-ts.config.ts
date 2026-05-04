import {defineConfig} from '@hey-api/openapi-ts';

export default defineConfig({
    input: '../api/openapi-bundle.gen.yaml',
    output: {
        path: './src/lib/api',
        postProcess: [
            'prettier',
        ],
    },
    plugins: [
        '@hey-api/schemas',
        {
            dates: false,
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
