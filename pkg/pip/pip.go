package pip

type pip struct{}

func New() pip {
	return pip{}
}

func (p pip) CalculateFlowImpact(oldPolicy, newPolicy Policy, flows []Flow) []Flow {
	// TODO: process the flows instead of returning them back unchanged
	return flows
}
