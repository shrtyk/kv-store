package metrics

import "google.golang.org/grpc/codes"

//go:generate mockery
type Metrics interface {
	HttpRequest(code int, method, path string, latency float64)
	GrpcRequest(code codes.Code, service, method string, latency float64)

	HttpPut(key string, duration float64)
	HttpDelete(key string, duration float64)
	HttpGet(key string, duration float64)

	GrpcPut(key string, duration float64)
	GrpcDelete(key string, duration float64)
	GrpcGet(key string, duration float64)
}
