package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	computeTargetAuto    = "auto"
	computeTargetLocal   = "local"
	computeTargetBigfoot = "bigfoot"
)

func normalizeComputeTarget(input string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "", computeTargetAuto:
		return computeTargetAuto, nil
	case computeTargetLocal:
		return computeTargetLocal, nil
	case computeTargetBigfoot:
		return computeTargetBigfoot, nil
	default:
		return "", errInvalidComputeTarget(input)
	}
}

func resolveComputeTarget(requested string, labels []string) string {
	if requested == computeTargetLocal || requested == computeTargetBigfoot {
		return requested
	}
	for _, label := range labels {
		if strings.EqualFold(strings.TrimSpace(label), "compute:bigfoot") {
			return computeTargetBigfoot
		}
	}
	return computeTargetLocal
}

func targetRigForCompute(rigName, computeTarget string) string {
	if computeTarget == computeTargetBigfoot {
		return computeTargetBigfoot
	}
	return rigName
}

func buildDispatchAudit(computeTarget string) (runID, routedHost, startedAt, resourceClass string) {
	runID = strings.TrimSpace(os.Getenv("GT_RUN"))
	if runID == "" {
		runID = uuid.New().String()
	}
	startedAt = time.Now().UTC().Format(time.RFC3339)
	switch computeTarget {
	case computeTargetBigfoot:
		routedHost = strings.TrimSpace(os.Getenv("GT_BIGFOOT_HOST"))
		if routedHost == "" {
			routedHost = computeTargetBigfoot
		}
		resourceClass = "burst"
	default:
		if host, err := os.Hostname(); err == nil && strings.TrimSpace(host) != "" {
			routedHost = host
		} else {
			routedHost = computeTargetLocal
		}
		resourceClass = "local"
	}
	return runID, routedHost, startedAt, resourceClass
}

type invalidComputeTargetError string

func (e invalidComputeTargetError) Error() string {
	return "invalid --compute-target value " + `"` + string(e) + `"` + ": must be auto, local, or bigfoot"
}

func errInvalidComputeTarget(v string) error {
	return invalidComputeTargetError(v)
}
