package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/smakimka/pam/internal/server/model"
	"github.com/smakimka/pam/internal/server/storage"
)

type AuthInterceptor struct {
	s storage.Storage
}

func NewAuthInterceptor(s storage.Storage) *AuthInterceptor {
	return &AuthInterceptor{s: s}
}

func (i *AuthInterceptor) Auth(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if info.FullMethod == "/PamServer/Authenticate" || info.FullMethod == "/PamServer/Register" {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	tokens := md.Get("auth-token")
	if len(tokens) != 1 {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}
	if tokens[0] == "" {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	user, err := i.s.GetUserByToken(ctx, tokens[0], time.Now())
	if err != nil {
		return nil, status.Error(codes.Internal, "unauthenticated")
	}

	tokenCtx := context.WithValue(ctx, model.UserID, user.ID)
	authCtx := context.WithValue(tokenCtx, model.AuthToken, tokens[0])
	return handler(authCtx, req)
}
