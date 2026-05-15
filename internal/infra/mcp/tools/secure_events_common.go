package tools

// secureEventsBaseFilter is the filter prefix applied to every runtime-events
// query the server makes on behalf of an LLM. It hides classes of events that
// are noisy at investigation time (benchmarks, posture findings, scanning
// activity) so that user-supplied filters target the runtime signal.
const secureEventsBaseFilter = `not originator in ("benchmarks","compliance","cloudsec","scanning","hostscanning")`

// composeSecureEventsFilter merges the user-supplied filter expression with
// the baseline. An empty userFilter returns the baseline unchanged.
func composeSecureEventsFilter(userFilter string) string {
	if userFilter == "" {
		return secureEventsBaseFilter
	}
	return secureEventsBaseFilter + " and " + userFilter
}

// secureEventsFilterDSL is the shared filter-expression description used by
// list_runtime_events, count_runtime_events, runtime_events_timeseries, and
// discover_runtime_event_field_values. Keeping the prose in one place lets the
// LLM apply identical filter intuition across all four tools.
const secureEventsFilterDSL = `Logical filter expression to select runtime security events.
Supports operators: =, !=, in, contains, starts with, exists.
Combine with and/or/not.
Key attributes include: severity (codes "0"-"7"), originator, sourceType, ruleName, rawEventCategory, engine, source, category, kubernetes.cluster.name, host.hostName, container.image.repo, container.image.tag, aws.accountId, azure.subscriptionId, gcp.projectId, policyId, trigger.

To find machine learning (ML) detections (e.g. crypto mining, anomalous logins), use engine or source filters:
- All ML events: 'engine = "machineLearning"'
- AWS ML detections: 'source = "agentless-aws-ml"'
- Okta ML detections: 'source = "agentless-okta-ml"'
- By category: 'category = "machine-learning"'

You can specify the severity of the events based on the following cases:
- high-severity: 'severity in ("0","1","2","3")'
- medium: 'severity in ("4","5")'
- low: 'severity in ("6")'
- info: 'severity in ("7")'
`
