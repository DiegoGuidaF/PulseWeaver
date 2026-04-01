import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { TopCountriesTable } from './TopCountriesTable';
import { createMockAccessLogCountryStats } from '@/test/mocks/data';

function renderTable(
    props?: Partial<React.ComponentProps<typeof TopCountriesTable>>,
) {
    return render(
        <MantineProvider>
            <TopCountriesTable
                data={props?.data ?? []}
                isLoading={props?.isLoading ?? false}
                metric={props?.metric ?? 'denied'}
                onCountryClick={props?.onCountryClick ?? vi.fn()}
            />
        </MantineProvider>,
    );
}

describe('TopCountriesTable', () => {
    it('renders skeleton when loading', () => {
        renderTable({ isLoading: true, data: undefined });
        // When loading: no table content, no empty state
        expect(screen.queryByText('No geographic data in this period')).not.toBeInTheDocument();
        expect(screen.queryByRole('table')).not.toBeInTheDocument();
    });

    it('renders empty state when no data', () => {
        renderTable({ data: [] });
        expect(
            screen.getByText('No geographic data in this period'),
        ).toBeInTheDocument();
    });

    it('renders heading', () => {
        renderTable({ data: [createMockAccessLogCountryStats()] });
        expect(screen.getByText('Top Countries')).toBeInTheDocument();
    });

    it('sorts rows by denied count descending', () => {
        const data = [
            createMockAccessLogCountryStats({ country_code: 'DE', country_name: 'Germany', denied: 5, allowed: 45, total: 50 }),
            createMockAccessLogCountryStats({ country_code: 'CN', country_name: 'China', denied: 70, allowed: 5, total: 75 }),
            createMockAccessLogCountryStats({ country_code: 'US', country_name: 'United States', denied: 20, allowed: 80, total: 100 }),
        ];
        renderTable({ data, metric: 'denied' });

        const rows = screen.getAllByRole('row').slice(1); // skip header
        expect(rows[0]).toHaveTextContent('China');
        expect(rows[1]).toHaveTextContent('United States');
        expect(rows[2]).toHaveTextContent('Germany');
    });

    it('sorts rows by total count descending when metric is total', () => {
        const data = [
            createMockAccessLogCountryStats({ country_code: 'DE', country_name: 'Germany', total: 50 }),
            createMockAccessLogCountryStats({ country_code: 'US', country_name: 'United States', total: 100 }),
            createMockAccessLogCountryStats({ country_code: 'CN', country_name: 'China', total: 75 }),
        ];
        renderTable({ data, metric: 'total' });

        const rows = screen.getAllByRole('row').slice(1);
        expect(rows[0]).toHaveTextContent('United States');
        expect(rows[1]).toHaveTextContent('China');
        expect(rows[2]).toHaveTextContent('Germany');
    });

    it('caps display at top 10 entries', () => {
        const data = Array.from({ length: 15 }, (_, i) =>
            createMockAccessLogCountryStats({
                country_code: `C${i}`,
                country_name: `Country ${i}`,
                denied: 100 - i,
                total: 100 - i,
                allowed: 0,
            }),
        );
        renderTable({ data });

        const rows = screen.getAllByRole('row').slice(1);
        expect(rows).toHaveLength(10);
    });

    it('calls onCountryClick when a row is clicked', () => {
        const onCountryClick = vi.fn();
        const data = [
            createMockAccessLogCountryStats({ country_code: 'US', country_name: 'United States' }),
        ];
        renderTable({ data, onCountryClick });

        // Text is rendered as "🇺🇸 United States" — use regex to match the name part
        fireEvent.click(screen.getByText(/United States/));
        expect(onCountryClick).toHaveBeenCalledWith('US');
    });

    it('displays rank numbers starting from 1', () => {
        const data = [
            createMockAccessLogCountryStats({ country_code: 'US', denied: 20 }),
            createMockAccessLogCountryStats({ country_code: 'DE', denied: 5 }),
        ];
        renderTable({ data });

        const rows = screen.getAllByRole('row').slice(1);
        expect(rows[0]).toHaveTextContent('1');
        expect(rows[1]).toHaveTextContent('2');
    });
});
