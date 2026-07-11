package anomaly

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Summarize renders the one-line human description of an anomaly from its
// kind and stored evidence. Evidence comes from JSON, so numbers are float64
// and any key may be missing; a missing key degrades the sentence, never errors.
func Summarize(kind Kind, evidence map[string]any) string {
	switch kind {
	case KindExpiredAccess:
		return summarizeExpiredAccess(evidence)
	case KindInvalidToken:
		return summarizeInvalidToken(evidence)
	case KindDenySpike:
		return summarizeDenySpike(evidence)
	case KindEntityDrift:
		return summarizeEntityDrift(evidence)
	case KindGeoDenied:
		return summarizeGeoDenied(evidence)
	case KindHostProbing:
		return summarizeHostProbing(evidence)
	case KindAddressChurn:
		return summarizeAddressChurn(evidence)
	case KindNewUserAgent:
		return summarizeNewUserAgent(evidence)
	case KindNewCountry:
		return summarizeNewCountry(evidence)
	case KindImpossibleTravel:
		return summarizeImpossibleTravel(evidence)
	default:
		return "Anomaly detected."
	}
}

func summarizeExpiredAccess(e map[string]any) string {
	if n, ok := intVal(e, "deny_count"); ok {
		return fmt.Sprintf("Denied %s after its address lease expired — the user may be silently locked out.", countPhrase(n, "time", "times"))
	}
	return "Denied after its address lease expired — the user may be silently locked out."
}

func summarizeInvalidToken(e map[string]any) string {
	var b strings.Builder
	if n, ok := intVal(e, "deny_count"); ok {
		b.WriteString(countPhrase(n, "request", "requests"))
	} else {
		b.WriteString("Requests")
	}
	b.WriteString(" with an invalid bearer token")
	if hosts, ok := stringSliceVal(e, "target_hosts"); ok {
		b.WriteString(" targeting ")
		b.WriteString(countPhrase(len(hosts), "host", "hosts"))
	}
	b.WriteString(" — the proxy token may be broken, or something else is calling the verify endpoint.")
	return b.String()
}

func summarizeDenySpike(e map[string]any) string {
	outcome, _ := stringVal(e, "outcome")
	noun := "requests"
	if outcome == "deny" {
		noun = "denials"
	}

	var b strings.Builder
	if observed, ok := intVal(e, "observed"); ok {
		fmt.Fprintf(&b, "%d %s", observed, noun)
	} else {
		b.WriteString("Unusual traffic volume")
	}
	b.WriteString(" in an hour")
	if series, ok := stringVal(e, "series"); ok {
		b.WriteString(" on ")
		b.WriteString(series)
	}
	if baseline, ok := floatVal(e, "baseline"); ok {
		b.WriteString(" vs a typical ")
		b.WriteString(formatBaseline(baseline))
	}
	if threshold, ok := intVal(e, "threshold"); ok {
		fmt.Fprintf(&b, " (threshold %d)", threshold)
	}
	b.WriteString(".")
	return b.String()
}

func summarizeEntityDrift(e map[string]any) string {
	kind, hasKind := stringVal(e, "entity_kind")
	name, hasName := stringVal(e, "entity_name")
	outcome, ok := stringVal(e, "outcome")
	if !ok {
		outcome = "request"
	}

	subject := "This entity"
	switch {
	case hasKind && hasName:
		subject = fmt.Sprintf("%s '%s'", capitalize(kind), name)
	case hasKind:
		subject = capitalize(kind)
	}

	var b strings.Builder
	b.WriteString(subject)
	b.WriteString(" saw ")
	if observed, ok := intVal(e, "observed"); ok {
		fmt.Fprintf(&b, "%d %s-requests", observed, outcome)
	} else {
		fmt.Fprintf(&b, "an unusual number of %s-requests", outcome)
	}
	b.WriteString(" in an hour")
	if baseline, ok := floatVal(e, "baseline"); ok {
		b.WriteString(" vs a typical ")
		b.WriteString(formatBaseline(baseline))
	}
	b.WriteString(".")
	return b.String()
}

func summarizeGeoDenied(e map[string]any) string {
	var b strings.Builder
	if n, ok := intVal(e, "deny_count"); ok {
		b.WriteString(countPhrase(n, "denial", "denials"))
	} else {
		b.WriteString("Denials")
	}
	if country, ok := stringVal(e, "country_name"); ok {
		b.WriteString(" from ")
		b.WriteString(country)
	} else {
		b.WriteString(" from an unrecognized country")
	}
	if org, ok := stringVal(e, "asn_org"); ok {
		b.WriteString(" (")
		b.WriteString(org)
		b.WriteString(")")
	}
	b.WriteString(", outside the expected countries.")
	return b.String()
}

func summarizeHostProbing(e map[string]any) string {
	var b strings.Builder
	b.WriteString("Denied on ")
	if hosts, ok := intVal(e, "distinct_hosts"); ok {
		b.WriteString(countPhrase(hosts, "distinct host", "distinct hosts"))
	} else {
		b.WriteString("multiple distinct hosts")
	}
	if n, ok := intVal(e, "deny_count"); ok {
		b.WriteString(" (")
		b.WriteString(countPhrase(n, "denial", "denials"))
		b.WriteString(")")
	}
	b.WriteString(" — fanning across services looks like probing.")
	return b.String()
}

func summarizeAddressChurn(e map[string]any) string {
	var b strings.Builder
	if n, ok := intVal(e, "new_addresses"); ok {
		b.WriteString(countPhrase(n, "new address", "new addresses"))
	} else {
		b.WriteString("New addresses")
	}
	b.WriteString(" registered within 24 h")
	if threshold, ok := intVal(e, "threshold"); ok {
		fmt.Fprintf(&b, " (threshold %d)", threshold)
	}
	b.WriteString(" — possible key sharing or spoofing.")
	return b.String()
}

func summarizeNewUserAgent(e map[string]any) string {
	if ua, ok := stringVal(e, "user_agent"); ok {
		return fmt.Sprintf("First time this device presents %q.", ua)
	}
	return "First time this device presents a new user agent."
}

func summarizeNewCountry(e map[string]any) string {
	if code, ok := stringVal(e, "country_code"); ok {
		return fmt.Sprintf("First activity from %s for this device.", code)
	}
	return "First activity from a new country for this device."
}

func summarizeImpossibleTravel(e map[string]any) string {
	signal, _ := stringVal(e, "signal")
	switch signal {
	case "concurrent_presence":
		if countries, ok := stringSliceVal(e, "countries"); ok {
			return fmt.Sprintf("Active in %s at the same time.", joinNatural(countries))
		}
	case "country_hop":
		from, hasFrom := stringVal(e, "from_country")
		to, hasTo := stringVal(e, "to_country")
		if hasFrom && hasTo {
			return fmt.Sprintf("Moved %s → %s faster than travel allows.", from, to)
		}
	}
	return "Impossible travel detected for this device."
}

// floatVal reads a numeric evidence value. Evidence is decoded from JSON, so
// every number surfaces as float64 regardless of what the detector stored.
func floatVal(e map[string]any, key string) (float64, bool) {
	f, ok := e[key].(float64)
	return f, ok
}

// intVal reads a numeric evidence value rounded to the nearest int. Every
// count/threshold field the detectors write is integral; rounding only
// guards against float imprecision surviving the JSON round trip.
func intVal(e map[string]any, key string) (int, bool) {
	f, ok := floatVal(e, key)
	if !ok {
		return 0, false
	}
	return int(math.Round(f)), true
}

func stringVal(e map[string]any, key string) (string, bool) {
	s, ok := e[key].(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// stringSliceVal reads a string-slice evidence value. JSON decoding yields
// []any with each element as a string; a directly assigned []string is also
// accepted for callers that build evidence maps in Go without a JSON round trip.
func stringSliceVal(e map[string]any, key string) ([]string, bool) {
	switch v := e[key].(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil, false
		}
		return out, true
	case []string:
		if len(v) == 0 {
			return nil, false
		}
		return v, true
	default:
		return nil, false
	}
}

// countPhrase renders "N singular" or "N plural" with proper pluralization —
// never the "(s)" shorthand.
func countPhrase(n int, singular, plural string) string {
	word := plural
	if n == 1 {
		word = singular
	}
	return fmt.Sprintf("%d %s", n, word)
}

// formatBaseline renders a possibly-fractional baseline with at most one
// decimal place, trimming a trailing ".0" so integral baselines read as
// plain integers.
func formatBaseline(f float64) string {
	s := strconv.FormatFloat(f, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// joinNatural renders a list of names in prose form: "a", "a and b", or
// "a, b, and c".
func joinNatural(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}
