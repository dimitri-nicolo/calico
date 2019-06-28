package collector

type DNSLogGetter interface {
	Get() []*DNSLog
}

type DNSLogAggregator interface {
	IncludeLabels(bool) DNSLogAggregator
	AggregateOver(AggregationKind) DNSLogAggregator
}

type DNSLogDispatcher interface {
	Initialize() error
	Dispatch([]*DNSLog) error
}
