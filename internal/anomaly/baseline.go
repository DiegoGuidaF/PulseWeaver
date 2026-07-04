package anomaly

// Preset holds the sensitivity-scaled thresholds the volume family evaluates
// against. The operator picks a preset name, never raw numbers, so a
// misconfiguration cannot land in a state they can't reason about; the numbers
// live here in code.
type Preset struct {
	// ProbingDistinctHosts is the distinct denied-host count over the trailing
	// window that flags a device for host_probing.
	ProbingDistinctHosts int
	// AddressChurnNewAddresses is the new-address count per device over the
	// trailing window that flags address_churn.
	AddressChurnNewAddresses int
}

// presetFor resolves a sensitivity name into its threshold set. Config
// validation already rejects unknown names, so the medium fallback is defensive
// rather than a supported path.
func presetFor(sensitivity string) Preset {
	switch sensitivity {
	case "low":
		return Preset{ProbingDistinctHosts: 8, AddressChurnNewAddresses: 15}
	case "high":
		return Preset{ProbingDistinctHosts: 3, AddressChurnNewAddresses: 6}
	default: // medium
		return Preset{ProbingDistinctHosts: 5, AddressChurnNewAddresses: 10}
	}
}
