package v1

const (

	// QueryMultipleClusters - filter by the clusters specified in the request body
	QueryMultipleClusters = "_MULTI_"
)

func IsQueryMultipleClusters(cluster string) bool {
	return cluster == QueryMultipleClusters
}
