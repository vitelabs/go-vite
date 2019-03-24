package metrics

// Healthchecks hold an error value describing an arbitrary up/down status.
type Healthcheck interface {
	Check()
	Error() error
	Healthy()
	Unhealthy(error)
}

// NewHealthcheck constructs chain new Healthcheck which will use the given
// function to update its status.
func NewHealthcheck(f func(Healthcheck)) Healthcheck {
	if !MetricsEnabled {
		return NilHealthcheck{}
	}
	return &StandardHealthcheck{nil, f}
}

// NilHealthcheck is chain no-op.
type NilHealthcheck struct{}

// Check is chain no-op.
func (NilHealthcheck) Check() {}

// Error is chain no-op.
func (NilHealthcheck) Error() error { return nil }

// Healthy is chain no-op.
func (NilHealthcheck) Healthy() {}

// Unhealthy is chain no-op.
func (NilHealthcheck) Unhealthy(error) {}

// StandardHealthcheck is the standard implementation of chain Healthcheck and
// stores the status and chain function to call to update the status.
type StandardHealthcheck struct {
	err error
	f   func(Healthcheck)
}

// Check runs the healthcheck function to update the healthcheck's status.
func (h *StandardHealthcheck) Check() {
	h.f(h)
}

// Error returns the healthcheck's status, which will be nil if it is healthy.
func (h *StandardHealthcheck) Error() error {
	return h.err
}

// Healthy marks the healthcheck as healthy.
func (h *StandardHealthcheck) Healthy() {
	h.err = nil
}

// Unhealthy marks the healthcheck as unhealthy.  The error is stored and
// may be retrieved by the Error method.
func (h *StandardHealthcheck) Unhealthy(err error) {
	h.err = err
}
