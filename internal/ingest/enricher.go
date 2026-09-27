package ingest

import (
	"regexp"
	"strings"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

var (
	nonAlphaNumHyphen = regexp.MustCompile(`[^a-z0-9\-]`)
)

func EnrichSpans(spans []domain.Span, orgID string) []domain.Span {
	now := time.Now()
	enriched := make([]domain.Span, len(spans))

	for i, s := range spans {
		s.OrgID = orgID
		s.ReceivedAt = now
		s.ServiceName = NormalizeServiceName(s.ServiceName)
		if s.Tags == nil {
			s.Tags = make(map[string]string)
		}
		enriched[i] = s
	}

	return enriched
}

func NormalizeServiceName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	// Strip anything that isn't alpha-numeric or hyphen
	name = nonAlphaNumHyphen.ReplaceAllString(name, "")
	name = strings.Trim(name, "-")
	return name
}
