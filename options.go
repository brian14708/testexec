package testexec

type options struct {
	Name string `json:"name,omitempty"`
}

type ProgramOption func(*options)

func WithName(name string) ProgramOption {
	return func(o *options) {
		o.Name = name
	}
}
