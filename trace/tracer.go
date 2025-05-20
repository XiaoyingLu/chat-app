package trace

import (
	"fmt"
	"io"
)

// Tracer is the interface that describes an object capable of tracing events throughout code
type Tracer interface {
	Trace(...interface{}) // accepts zero or more arguments of any type
}

// tracer is unexported, because it is an implementation of Tracer
type tracer struct {
	out io.Writer
}
func (t *tracer) Trace(a ...interface{}) {
	fmt.Fprint(t.out, a...)
	fmt.Fprintln(t.out)
}

func New(w io.Writer) Tracer {
	return &tracer{out: w}
}

type nilTracer struct {}
func (t *nilTracer) Trace(a ...interface{}) {}

// Off creates a Tracer that will ignore calls to Trace
func Off() Tracer {
	return &nilTracer{}
}
