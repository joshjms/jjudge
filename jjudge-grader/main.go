package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/jjudge-oj/grader/api/graderpb"
	"github.com/jjudge-oj/grader/config"
	"google.golang.org/grpc"
)

type server struct {
	graderpb.UnimplementedGraderServer
}

func gradeStringMatching(expected, output string) bool {
	return expected == output
}

func gradeTokenMatching(expected, output string) bool {
	expectedTokens := strings.Split(expected, " ")
	outputTokens := strings.Split(output, " ")

	if len(expectedTokens) != len(outputTokens) {
		return false
	}

	for i := range expectedTokens {
		if expectedTokens[i] != outputTokens[i] {
			return false
		}
	}

	return true
}

func (s *server) Grade(ctx context.Context, req *graderpb.GraderRequest) (*graderpb.GraderResponse, error) {
	switch req.UseGrader {
	case "string":
		return &graderpb.GraderResponse{
			Ok: gradeStringMatching(req.ExpectedOutput, req.Output),
		}, nil
	case "token":
		return &graderpb.GraderResponse{
			Ok: gradeTokenMatching(req.ExpectedOutput, req.Output),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported grader type: %s", req.GetUseGrader())
	}
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
