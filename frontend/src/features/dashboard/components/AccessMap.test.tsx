import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { AccessMap } from './AccessMap';
import { createMockAccessLogCountryStats } from '@/test/mocks/data';
import type { AccessLogCountryStats } from '@/lib/api/types.gen';

const noopColorFn = () => '#555555';

function renderMap(props?: Partial<React.ComponentProps<typeof AccessMap>>) {
    const lookup = new Map<string, AccessLogCountryStats>();
    return render(
        <MantineProvider>
            <AccessMap
                data={props?.data ?? []}
                isLoading={props?.isLoading ?? false}
                metric={props?.metric ?? "denied"}
                onMetricChange={props?.onMetricChange ?? vi.fn()}
                colorFn={props?.colorFn ?? noopColorFn}
                lookup={props?.lookup ?? lookup}
                onCountryClick={props?.onCountryClick ?? vi.fn()}
            />
        </MantineProvider>,
    );
}

describe('AccessMap', () => {
    it('renders skeleton when loading', () => {
        renderMap({ isLoading: true, data: undefined });
        // When loading: no SVG map and no empty state
        expect(screen.queryByRole('img')).not.toBeInTheDocument();
        expect(screen.queryByText('No geographic data in this period')).not.toBeInTheDocument();
    });

    it('renders empty state when data is empty', () => {
        renderMap({ data: [] });
        expect(
            screen.getByText('No geographic data in this period'),
        ).toBeInTheDocument();
    });

    it('renders empty state when data is undefined', () => {
        renderMap({ data: undefined });
        expect(
            screen.getByText('No geographic data in this period'),
        ).toBeInTheDocument();
    });

    it('renders SVG world map when data is provided', () => {
        const data = [createMockAccessLogCountryStats()];
        const lookup = new Map([['US', data[0]]]);
        renderMap({ data, lookup });

        const svg = screen.getByRole('img');
        expect(svg).toBeInTheDocument();
        // Should render country paths (world-atlas 110m has ~177 countries)
        const paths = document.querySelectorAll('svg path');
        expect(paths.length).toBeGreaterThan(100);
    });

    it('calls onCountryClick when clicking a country with data', () => {
        const onCountryClick = vi.fn();
        const data = [createMockAccessLogCountryStats({ country_code: 'US' })];
        const lookup = new Map([['US', data[0]]]);

        renderMap({ data, lookup, onCountryClick });

        // Find and click a path with pointer cursor (countries with data)
        const paths = Array.from(document.querySelectorAll('svg path'));
        const clickable = paths.find(
            (p) => (p as HTMLElement).style.cursor === 'pointer',
        );
        expect(clickable).toBeDefined();
        fireEvent.click(clickable!);
        expect(onCountryClick).toHaveBeenCalledWith('US');
    });

    it('renders heading', () => {
        const data = [createMockAccessLogCountryStats()];
        const lookup = new Map([['US', data[0]]]);
        renderMap({ data, lookup });
        expect(screen.getByText('Access Map')).toBeInTheDocument();
    });
});
