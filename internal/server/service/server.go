package service

import (
	"github.com/smakimka/pam/internal/protobuf/pamserver"
	"github.com/smakimka/pam/internal/server/interceptors"
	"github.com/smakimka/pam/internal/server/storage"
	"google.golang.org/grpc"
)

func NewServer(s storage.Storage, expiryTime int) *grpc.Server {
	interceptor := interceptors.NewAuthInterceptor(s)

	service := newPamSerice(s, expiryTime)
	server := grpc.NewServer(grpc.UnaryInterceptor(interceptor.Auth))

	pamserver.RegisterPamServerServer(server, service)

	return server
}
