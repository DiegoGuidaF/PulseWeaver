import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils";
import { GeoCell } from "./GeoCell";

describe("GeoCell", () => {
  it("renders nothing when geo is absent", () => {
    renderWithProviders(
      <div data-testid="wrap">
        <GeoCell geo={null} />
      </div>,
    );
    expect(screen.getByTestId("wrap")).toBeEmptyDOMElement();
  });

  it("renders nothing when geo carries no displayable value", () => {
    renderWithProviders(
      <div data-testid="wrap">
        <GeoCell geo={{ continent_code: "EU" }} />
      </div>,
    );
    expect(screen.getByTestId("wrap")).toBeEmptyDOMElement();
  });

  it("renders the country code with flag", () => {
    renderWithProviders(<GeoCell geo={{ country_code: "DE", country_name: "Germany" }} />);
    // Flag emoji is prefixed; assert the code is shown.
    expect(screen.getByText(/DE/)).toBeInTheDocument();
  });

  it("shows the ASN org on a secondary line when showAsn is set", () => {
    renderWithProviders(
      <GeoCell geo={{ country_code: "US", asn: 13335, asn_org: "Cloudflare, Inc." }} showAsn />,
    );
    expect(screen.getByText("Cloudflare, Inc.")).toBeInTheDocument();
  });

  it("falls back to the ASN org as primary text when no country is present", () => {
    renderWithProviders(<GeoCell geo={{ asn: 13335, asn_org: "Cloudflare, Inc." }} />);
    expect(screen.getByText("Cloudflare, Inc.")).toBeInTheDocument();
  });
});
