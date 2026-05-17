package server

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// registerHealthServer registers a gRPC health server on s, setting SERVING status
// for the empty service name and for each registered service descriptor.
func registerHealthServer(s *grpc.Server) *health.Server {
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	for name := range s.GetServiceInfo() {
		hs.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)
	}
	healthpb.RegisterHealthServer(s, hs)
	return hs
}

// SetHealthStatus updates the serving status for a specific service name.
// Use an empty string ("") to set the overall server status.
// See healthpb.HealthCheckResponse_ServingStatus for valid values.
func (s *Server) SetHealthStatus(service string, status healthpb.HealthCheckResponse_ServingStatus) {
	s.grpcServer.healthServer.SetServingStatus(service, status)
}
