package util

func Sptr(s string) *string {
	sCopy := s
	return &sCopy
}
