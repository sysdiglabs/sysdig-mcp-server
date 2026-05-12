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

func mergeLimit(p sysdig.GetQueryV1Params, limit int) sysdig.GetQueryV1Params {
	lq := sysdig.LimitQuery(limit)
	p.Limit = &lq
	return p
}
