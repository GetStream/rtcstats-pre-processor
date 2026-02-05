package transform

import (
	"regexp"
	"strings"
)

// SFU hostname pattern: 0-sfu-dpk-frankfurt-vp1-54d1dc529306.stream-io-video.com
var sfuHostnamePattern = regexp.MustCompile(`^(\d+)-sfu-[a-z]+-([a-z]+-[a-z0-9]+)-[a-f0-9]+\.stream-io-video\.com$`)

// CompressScope compresses scope strings
// Short scopes like "0-pub", "0-sub" are kept as-is
// SFU hostnames are compressed to "sfu:<region>"
func CompressScope(scope *string) string {
	if scope == nil {
		return ""
	}

	s := *scope

	// Keep short scopes as-is
	if len(s) <= 10 || strings.HasSuffix(s, "-pub") || strings.HasSuffix(s, "-sub") {
		return s
	}

	// Try to match SFU hostname pattern
	matches := sfuHostnamePattern.FindStringSubmatch(s)
	if len(matches) == 3 {
		// matches[1] = prefix number, matches[2] = region-cluster
		return "sfu:" + matches[2]
	}

	// For other long hostnames, try to extract region
	if strings.Contains(s, ".stream-io-video.com") {
		// Try simpler pattern for any stream-io hostname
		parts := strings.Split(s, "-")
		if len(parts) >= 3 {
			// Look for region pattern (city-something)
			for i := 1; i < len(parts)-1; i++ {
				if isRegion(parts[i]) {
					region := parts[i]
					if i+1 < len(parts) && isCluster(parts[i+1]) {
						region += "-" + parts[i+1]
					}
					return "sfu:" + region
				}
			}
		}
	}

	// Truncate very long scopes
	if len(s) > 40 {
		return s[:40] + "..."
	}

	return s
}

// Known regions
var regions = map[string]bool{
	"frankfurt": true,
	"london":    true,
	"paris":     true,
	"amsterdam": true,
	"newyork":   true,
	"chicago":   true,
	"losangeles": true,
	"singapore": true,
	"tokyo":     true,
	"sydney":    true,
	"mumbai":    true,
	"saopaulo":  true,
}

func isRegion(s string) bool {
	return regions[strings.ToLower(s)]
}

func isCluster(s string) bool {
	// Cluster patterns like "vp1", "vp2", etc.
	if len(s) >= 2 && len(s) <= 4 {
		return true
	}
	return false
}
