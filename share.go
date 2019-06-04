package share

const (
	masterTimeout = 30 // seconds
	keyTTL        = 29 // seconds

	cmdReportWorkerName = "CMD_REPORT_WORKER_NAME"
	cmdStartWork        = "CMD_START_WORK"
)

type ShareOption interface {
	Apply(*share)
}

type withName string

func (w withName) Apply(o *share) {
	o.name = string(w)
}

func WithName(v string) ShareOption {
	return withName(v)
}

type share struct {
	name string
}

func (s share) Name() string { return s.name }

func New(o ...ShareOption) (*share, error) {
	s := &share{}
	for _, opt := range o {
		opt.Apply(s)
	}

	return s, nil
}
