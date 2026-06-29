package responsewrapper

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ResponseStatus represents the status part of the response
type ResponseStatus struct {
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	AlertMessage string `json:"alert_message"`
}

// StandardResponse represents the standardized response format
type StandardResponse struct {
	Status ResponseStatus `json:"status"`
	Data   interface{}    `json:"data"`
}

var (
	marshaler = &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames:     true,
			EmitUnpopulated:   true,
			EmitDefaultValues: true,
		},
	}
)

// ResponseWrapper is a middleware that wraps gRPC responses in a standard format
func ResponseWrapper() runtime.ServeMuxOption {
	return runtime.WithForwardResponseRewriter(func(ctx context.Context, resp proto.Message) (any, error) {

		buf, err := marshaler.Marshal(resp)
		if err != nil {
			return nil, err
		}
		// Create standard response wrapper
		standardResp := &StandardResponse{
			Status: ResponseStatus{
				ErrorCode:    0,
				ErrorMessage: "Success",
			},
			Data: json.RawMessage(buf),
		}
		// Set content type
		// w.Header().Set("Content-Type", "application/json")

		// Encode response
		return standardResp, nil
	})
}

func HTTPStatusFromCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return 499
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		// Note, this deliberately doesn't translate to the similarly named '412 Precondition Failed' HTTP response status.
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	default:
		// OK but
		return http.StatusOK
	}
}

// ErrorHandler handles gRPC errors and converts them to standard format
func ErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	// Get gRPC status
	st := status.Convert(err)

	// Determine HTTP status code
	httpStatus := HTTPStatusFromCode(st.Code())

	// Create error response
	errorResp := StandardResponse{
		Status: ResponseStatus{
			ErrorCode:    int(st.Code()),
			ErrorMessage: st.Message(),
		},
		Data: nil,
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	// if st.code() is not standard, set httpStatus to 200 and set status.error_code to st.code()
	w.WriteHeader(httpStatus)

	buf, err := marshaler.Marshal(errorResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err = w.Write(buf); err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		grpclog.Errorf("Failed to write response: %v", err)
	}
}
