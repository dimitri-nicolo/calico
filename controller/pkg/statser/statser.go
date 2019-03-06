package statser

type Statser interface {
	Status() *Status
	SuccessfulSync()
	SuccessfulSearch()
	Error(string, error)
	ClearError(string)
}

func NewStatser() Statser {
	return &Status{}
}

