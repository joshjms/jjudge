package main

import (
	"context"
	"fmt"
	"net"

	"github.com/jjudge-oj/grader/api/graderpb"
	"github.com/jjudge-oj/grader/config"
	"google.golang.org/grpc"
)

type server struct {
	graderpb.UnimplementedGraderServer
}

func (s *server) Grade(ctx context.Context, req *graderpb.GraderRequest) (*graderpb.GraderResponse, error) {
	ok := req.Output == req.ExpectedOutput
	return &graderpb.GraderResponse{
		Ok: ok,
	}, nil
}

func main() {
	cfg := config.LoadConfig()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		fmt.Printf("failed to listen: %v\n", err)
		return
	}

	grpcServer := grpc.NewServer()
	graderpb.RegisterGraderServer(grpcServer, &server{})

	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v\n", err)
	}
}
