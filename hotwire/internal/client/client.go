package client

import (
	"context"
	"fmt"
	"time"

	hotwirev1 "github.com/aamoghS/sideprojects/hotwire/proto/hotwire/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Dial(ctx context.Context, addr string, opts ...grpc.DialOption) (*grpc.ClientConn, hotwirev1.ControlPlaneClient, error) {
	base := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.NewClient(addr, append(base, opts...)...)
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return conn, hotwirev1.NewControlPlaneClient(conn), nil
}

func SubscribeWeights(ctx context.Context, c hotwirev1.ControlPlaneClient) (hotwirev1.ControlPlane_SubscribeWeightsClient, error) {
	return c.SubscribeWeights(ctx, &hotwirev1.SubscribeRequest{})
}

func ListBackends(ctx context.Context, c hotwirev1.ControlPlaneClient) (*hotwirev1.ListBackendsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return c.ListBackends(ctx, &hotwirev1.ListBackendsRequest{})
}

func ReportMetrics(ctx context.Context, c hotwirev1.ControlPlaneClient) (hotwirev1.ControlPlane_ReportMetricsClient, error) {
	return c.ReportMetrics(ctx)
}
