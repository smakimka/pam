package pamclient

import (
	"context"
	"errors"

	"github.com/smakimka/pam/internal/protobuf/pamserver"
	"google.golang.org/grpc/metadata"
)

var ErrUnauthenticated = errors.New("unauthenticated")
var ErrWrongCredentials = errors.New("wromg credentials")
var ErrUsernameIsTaken = errors.New("this username is taken")
var ErrDataDoesNotExist = errors.New("this data doesn't exist'")

type GetResponse struct {
	Kind int
	Data []byte
}

type PamGRPCClient struct {
	client pamserver.PamServerClient
}

func NewGRPCClient(client pamserver.PamServerClient) *PamGRPCClient {
	return &PamGRPCClient{client: client}
}

func (c *PamGRPCClient) Register(ctx context.Context, username string, pwd string) (string, error) {
	resp, err := c.client.Register(ctx, &pamserver.AuthData{Username: username, Pwd: pwd})

	if err != nil {
		return "", err
	}

	if resp.Error == "user already exists" {
		return "", ErrUsernameIsTaken
	}

	return resp.Token, err
}

func (c *PamGRPCClient) Auth(ctx context.Context, username string, pwd string) (string, error) {
	resp, err := c.client.Authenticate(ctx, &pamserver.AuthData{Username: username, Pwd: pwd})

	if err != nil {
		if err.Error() == "rpc error: code = NotFound desc = wrong username or password" {
			return "", ErrWrongCredentials
		}
		return "", err
	}

	return resp.Token, err
}

func (c *PamGRPCClient) Upload(ctx context.Context, authToken string, name string, kind int, data []byte) error {
	md := metadata.New(map[string]string{"auth-token": authToken})
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.client.Upload(ctx, &pamserver.UploadData{Name: name, Type: int32(kind), Data: data})

	if err != nil {
		if err.Error() == "rpc error: code = Internal desc = unauthenticated" {
			return ErrUnauthenticated
		}
		return err
	}

	return nil
}

func (c *PamGRPCClient) Get(ctx context.Context, authToken string, name string) (*GetResponse, error) {
	md := metadata.New(map[string]string{"auth-token": authToken})
	ctx = metadata.NewOutgoingContext(ctx, md)

	data, err := c.client.Get(ctx, &pamserver.GetData{Name: name})
	if err != nil {
		if err.Error() == "rpc error: code = Internal desc = unauthenticated" {
			return nil, ErrUnauthenticated
		}

		if err.Error() == "rpc error: code = NotFound desc = this data does not exist" {
			return nil, ErrDataDoesNotExist
		}
		return nil, err
	}

	return &GetResponse{Kind: int(data.Kind), Data: data.Data}, nil
}

func (c *PamGRPCClient) List(ctx context.Context, authToken string) ([]string, error) {
	md := metadata.New(map[string]string{"auth-token": authToken})
	ctx = metadata.NewOutgoingContext(ctx, md)

	names, err := c.client.GetNames(ctx, &pamserver.GetDataNames{})
	if err != nil {
		if err.Error() == "rpc error: code = Internal desc = unauthenticated" {
			return nil, ErrUnauthenticated
		}

		return nil, err
	}

	return names.Names, nil
}
