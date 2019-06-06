package pip

type PIP interface {
	CalculateFlowImpact(oldPolicy, newPolicy Policy, flows []Flow) []Flow
}
