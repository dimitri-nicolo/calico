package puller

type TemporaryError string

func (e TemporaryError) Error() string {
	return string(e)
}

func (e TemporaryError) Temporary() bool {
	return true
}
