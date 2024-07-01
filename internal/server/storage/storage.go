package storage

import (
	"context"
	"time"

	"github.com/smakimka/pam/internal/server/model"
)

type Storage interface {
	Init(ctx context.Context) error

	GetUser(ctx context.Context, username string) (*model.UserData, error)
	GetData(ctx context.Context, userID int, name string) (*model.Data, error)
	GetDataNames(ctx context.Context, userID int) ([]string, error)
	GetUserByToken(ctx context.Context, token string, now time.Time) (*model.UserData, error)

	CreateUser(ctx context.Context, username string, pwd []byte) (int, error)
	CreateAuthToken(ctx context.Context, userID int, value string, expiry time.Time) (int, error)

	UpdateTokenExpiry(ctx context.Context, token string, newExpiry time.Time) error
	UpsertData(ctx context.Context, userID int, name string, kind int, data []byte) (int, error)
}
