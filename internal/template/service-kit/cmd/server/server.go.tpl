package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	pb "github.com/minhgiang16983/service-kit/pb"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

// RunGrpcGateway starts gRPC Gateway
func RunGrpcGateway(ctx context.Context, grpcPort int) (*http.ServeMux, error) {
	gatewayMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
		runtime.WithIncomingHeaderMatcher(CustomerHeaderMatcher),
	)
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Init routing
	r := http.NewServeMux()
	r.Handle("/", gatewayMux)
	r.Handle("/metrics", promhttp.Handler())

	err := pb.RegisterServiceKitHandlerFromEndpoint(ctx, gatewayMux, fmt.Sprintf(":%d", grpcPort), dialOpts)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func CustomerHeaderMatcher(key string) (string, bool) {
	keyLower := strings.ToLower(key)

	// Allow all headers starting with x-
	if strings.HasPrefix(keyLower, "x-") {
		return keyLower, true
	}

	// For standard headers, use default fallback
	return runtime.DefaultHeaderMatcher(key)
}
