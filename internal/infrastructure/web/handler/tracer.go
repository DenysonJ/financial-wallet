package handler

// handlerTracer is the OpenTelemetry tracer name used by every HTTP handler
// when starting a span (otel.Tracer(handlerTracer).Start(...)).
const handlerTracer = "http-handler"
