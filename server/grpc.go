package main

import (
	"context"
	"errors"
	"net"

	pb "benchmark/server/eventpb"
	"google.golang.org/grpc"
)

type grpcServer struct {
	pb.UnimplementedEventServiceServer
}

func (s *grpcServer) LogEvent(ctx context.Context, req *pb.LogRequest) (*pb.LogResponse, error) {
	if req.EventId == "" || req.UserId == "" {
		return nil, errors.New("missing fields")
	}
	return &pb.LogResponse{Status: "success"}, nil
}

func StartGRPCServer(addr string) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}

	s := grpc.NewServer()
	pb.RegisterEventServiceServer(s, &grpcServer{})

	return s, lis, nil
}
