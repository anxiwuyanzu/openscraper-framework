package trace

import (
	"errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Span struct {
	span trace.Span
}

func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
	for _, attr := range attrs {
		s.span.SetAttributes(attr)
	}
}

func (s *Span) AddEvent(name string, options ...trace.EventOption) {
	s.span.AddEvent(name, options...)
}

func (s *Span) SetError(errMsg string) {
	s.span.RecordError(errors.New(errMsg))
	s.span.SetStatus(codes.Error, errMsg)
}

func (s *Span) End() {
	s.span.End()
}
