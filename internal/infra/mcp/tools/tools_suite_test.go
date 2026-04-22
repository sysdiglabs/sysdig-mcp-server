package tools_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

func TestTools(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tools Suite")
}

// newWindowedQueryParams constructs the GetQueryV1Params value that a windowed tool
// invocation is expected to produce: Query string, Time = end.Unix() via FromQueryTime1,
// and a 60s Timeout.
func newWindowedQueryParams(query string, end time.Time) sysdig.GetQueryV1Params {
	var qt sysdig.Time
	Expect(qt.FromQueryTime1(end.Unix())).To(Succeed())
	timeout := sysdig.Timeout("60s")
	return sysdig.GetQueryV1Params{
		Query:   query,
		Time:    &qt,
		Timeout: &timeout,
	}
}

// mergeLimit attaches a Limit field to an existing GetQueryV1Params value.
// Used by tools that set params.Limit (memory_*, count_pods, underutilized_*).
func mergeLimit(p sysdig.GetQueryV1Params, limit int) sysdig.GetQueryV1Params {
	lq := sysdig.LimitQuery(limit)
	p.Limit = &lq
	return p
}
