package gapi

import (
	"context"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	grpcGatewayUserAgentHeader = "grpcgateway-user-agent"
	grpcGatewayClientIPHeader  = "x-forwarded-for"
	grpcUserAgentHeader        = "user-agent"
)

type MetaData struct {
	UserAgent string
	ClientIP  string
}

func (server *Server) extractMetaData(ctx context.Context) (metaData *MetaData) {
	metaData = &MetaData{}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgents := md.Get(grpcGatewayUserAgentHeader); len(userAgents) > 0 {
			metaData.UserAgent = userAgents[0]
		}
		if clientIPs := md.Get(grpcGatewayClientIPHeader); len(clientIPs) > 0 {
			metaData.ClientIP = clientIPs[0]
		}

		if userAgents := md.Get(grpcUserAgentHeader); len(userAgents) > 0 {
			metaData.UserAgent = userAgents[0]
		}
	}

	if p, ok := peer.FromContext(ctx); ok {
		metaData.ClientIP = p.Addr.String()
	}

	return
}
