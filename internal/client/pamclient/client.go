package pamclient

import "context"

type PamClient interface {
	Register(ctx context.Context, username string, pwd string) (string, error)
	Auth(ctx context.Context, username string, pwd string) (string, error)
	Get(ctx context.Context, authToken string, name string) (*GetResponse, error)
	List(ctx context.Context, authToken string) ([]string, error)
	Upload(ctx context.Context, authToken string, name string, kind int, data []byte) error
}
