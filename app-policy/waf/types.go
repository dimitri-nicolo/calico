package waf

// WAF Error structures
type WAFError struct {
	Disruption WAFIntervention
	Msg        string
}

type WAFIntervention struct {
	Status int32
	Log    string
	URL    string
}

func (e WAFError) Error() string {
	return e.Msg
}
