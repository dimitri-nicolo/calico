package feed

type Feed interface {
	Name() string
	Namespace() string
}

type feed struct {
	name      string
	namespace string
}

func NewFeed(name, namespace string) Feed {
	return &feed{name, namespace}
}

func (f *feed) Name() string {
	return f.name
}

func (f *feed) Namespace() string {
	return f.namespace
}

type IPSet []string
