package attributes

import (
	"context"
	"net"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// RequestAttributes represents unified request attributes extracted from headers
type RequestAttributes struct {
	IP        string            `json:"ip"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	UserID    string            `json:"user_id"`
	StaffID   string            `json:"staff_id"`
	DeviceID  string            `json:"device_id"`
	Headers   map[string]string `json:"headers"`
	RequestID string            `json:"request_id"`
}

func (r *RequestAttributes) Get(key string) string {
	key = strings.ToLower(key)
	return r.Headers[key]
}

// AttributeExtractor interface for extracting request attributes
type AttributeExtractor interface {
	Extract(ctx context.Context) (*RequestAttributes, error)
}

// HTTPAttributeExtractor extracts attributes from HTTP requests
type HTTPAttributeExtractor struct {
	request *http.Request
}

// NewHTTPAttributeExtractor creates a new HTTP attribute extractor
func NewHTTPAttributeExtractor(r *http.Request) *HTTPAttributeExtractor {
	return &HTTPAttributeExtractor{
		request: r,
	}
}

// Extract extracts request attributes from HTTP request
func (h *HTTPAttributeExtractor) Extract(ctx context.Context) (*RequestAttributes, error) {
	attrs := &RequestAttributes{
		Headers: make(map[string]string),
	}

	// Extract IP address
	attrs.IP = h.getIP()

	// Extract method and path
	attrs.Method = h.request.Method
	attrs.Path = h.request.URL.Path

	// Extract headers
	for key, values := range h.request.Header {
		if len(values) > 0 {
			attrs.Headers[strings.ToLower(key)] = values[0]
		}
	}

	// Extract specific headers
	attrs.UserID = h.getHeaderValue("x-user-id")
	attrs.StaffID = h.getHeaderValue("x-staff-id")
	attrs.DeviceID = h.getHeaderValue("x-device-id")
	attrs.RequestID = h.getRequestID()

	return attrs, nil
}

// getIP extracts the real IP address from various headers
func (h *HTTPAttributeExtractor) getIP() string {
	// Check X-Forwarded-For first (for proxy scenarios)
	if forwardedFor := h.request.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP
	if realIP := h.request.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Check X-Client-IP
	if clientIP := h.request.Header.Get("X-Client-IP"); clientIP != "" {
		return clientIP
	}

	// Fallback to RemoteAddr
	if h.request.RemoteAddr != "" {
		// Remove port if present
		if colonIndex := strings.LastIndex(h.request.RemoteAddr, ":"); colonIndex != -1 {
			return h.request.RemoteAddr[:colonIndex]
		}
		return h.request.RemoteAddr
	}

	return "unknown"
}

func (h *HTTPAttributeExtractor) getRequestID() string {
	request := h.getHeaderValue("x-kong-request-id")
	if request == "" {
		request = h.getHeaderValue("x-request-id")
	}
	return request
}

// getHeaderValue safely extracts a header value
func (h *HTTPAttributeExtractor) getHeaderValue(key string) string {
	key = strings.ToLower(key)
	return h.request.Header.Get(key)
}

// GRPCAttributeExtractor extracts attributes from gRPC metadata
type GRPCAttributeExtractor struct {
	md   metadata.MD
	info *grpc.UnaryServerInfo
}

// NewGRPCAttributeExtractor creates a new gRPC attribute extractor
func NewGRPCAttributeExtractor(ctx context.Context, info *grpc.UnaryServerInfo) *GRPCAttributeExtractor {
	md, _ := metadata.FromIncomingContext(ctx)
	return &GRPCAttributeExtractor{
		md:   md,
		info: info,
	}
}

// Extract extracts request attributes from gRPC metadata
func (g *GRPCAttributeExtractor) Extract(ctx context.Context) (*RequestAttributes, error) {
	attrs := &RequestAttributes{
		Headers: make(map[string]string),
	}

	// Extract IP address
	attrs.IP = g.getIP(ctx)

	// Extract method and path from gRPC info
	attrs.Method = "gRPC"
	if g.info != nil {
		attrs.Path = g.info.FullMethod
	} else {
		attrs.Path = "unknown"
	}

	// Extract headers from metadata
	for key, values := range g.md {
		if len(values) > 0 {
			attrs.Headers[strings.ToLower(key)] = values[0]
		}
	}

	// Extract specific headers
	attrs.UserID = g.getMetadataValue("x-user-id")
	attrs.StaffID = g.getMetadataValue("x-staff-id")
	attrs.DeviceID = g.getMetadataValue("x-device-id")

	return attrs, nil
}

// getIP extracts the IP address from metadata
func (g *GRPCAttributeExtractor) getIP(ctx context.Context) string {
	// Check X-Forwarded-For first
	if ips := g.md.Get("x-forwarded-for"); len(ips) > 0 && ips[0] != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ipList := strings.Split(ips[0], ",")
		if len(ipList) > 0 {
			return strings.TrimSpace(ipList[0])
		}
	}

	// Check X-Real-IP
	if realIPs := g.md.Get("x-real-ip"); len(realIPs) > 0 && realIPs[0] != "" {
		return realIPs[0]
	}

	// Check X-Client-IP
	if clientIPs := g.md.Get("x-client-ip"); len(clientIPs) > 0 && clientIPs[0] != "" {
		return clientIPs[0]
	}

	// Try to get IP from peer context
	if p, ok := peer.FromContext(ctx); ok {
		if addr := p.Addr; addr != nil {
			if tcpAddr, ok := addr.(*net.TCPAddr); ok {
				return tcpAddr.IP.String()
			}
			// Fallback to string representation
			return addr.String()
		}
	}

	return "unknown"
}

// getMetadataValue safely extracts a metadata value
func (g *GRPCAttributeExtractor) getMetadataValue(key string) string {
	key = strings.ToLower(key)
	values := g.md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// Context keys for storing extracted attributes
type contextKey string

const (
	RequestAttributesKey contextKey = "request_attributes"
)

// WithRequestAttributes adds request attributes to context
func WithRequestAttributes(ctx context.Context, attrs *RequestAttributes) context.Context {
	return context.WithValue(ctx, RequestAttributesKey, attrs)
}

// GetRequestAttributes retrieves request attributes from context
func GetFromContext(ctx context.Context) (*RequestAttributes, bool) {
	attrs, ok := ctx.Value(RequestAttributesKey).(*RequestAttributes)
	return attrs, ok
}
