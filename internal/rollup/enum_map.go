package rollup

import "github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"

// attributionKindFromAPI converts the API entity-kind parameter to the domain
// AttributionKind used to key attributionSpecs. The exhaustive linter fails
// this switch when a new AttributionEntityKind value is added to the API
// without a corresponding case (and spec entry) here.
func attributionKindFromAPI(k httpapi.AttributionEntityKind) AttributionKind {
	switch k {
	case httpapi.AttributionEntityKindPolicy:
		return AttributionKindPolicy
	case httpapi.AttributionEntityKindUser:
		return AttributionKindUser
	case httpapi.AttributionEntityKindDevice:
		return AttributionKindDevice
	default:
		return AttributionKind(k)
	}
}
