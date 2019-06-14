package pip

type PIP interface {
	CalculateFlowImpact(npcs []NetworkPolicyChange, flows []Flow) []Flow
}
