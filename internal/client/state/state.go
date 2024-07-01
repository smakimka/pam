package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/adrg/xdg"
	"github.com/smakimka/pam/internal/client/pamclient"
	"golang.org/x/term"
)

type State struct {
	dataFile   *os.File
	client     pamclient.PamClient
	ServerAddr string `json:"server_addr"`
	AuthToken  string `json:"auth_token"`
}

func Open() (*State, error) {
	cfg := &State{}

	filePath, err := xdg.DataFile("pam/pam.data")
	if err != nil {
		return cfg, err
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return cfg, err
	}
	cfg.dataFile = file

	data, err := io.ReadAll(file)
	if err != nil {
		return cfg, err
	}

	if len(data) != 0 {
		err = json.Unmarshal(data, cfg)
		if err != nil {
			return cfg, err
		}
	}

	return cfg, nil
}

func (s *State) SetClient(c pamclient.PamClient) {
	s.client = c
}

func (s *State) Close() {
	if s.dataFile == nil {
		return
	}
	defer s.dataFile.Close()

	data, err := json.Marshal(s)
	if err != nil {
		return
	}

	_, err = s.dataFile.Seek(0, 0)
	if err != nil {
		return
	}
	err = s.dataFile.Truncate(0)
	if err != nil {
		return
	}

	_, err = s.dataFile.Write(data)
	if err != nil {
		return
	}
}

func (s *State) ReadServerAddr() error {
	var serverAddr string

	fmt.Print("Enter server addr: ")
	_, err := fmt.Scanln(&serverAddr)
	if err != nil {
		return err
	}

	s.ServerAddr = serverAddr

	return nil
}

func (s *State) collectUsernameAndPwd() ([]string, error) {
	var username string
	fmt.Print("Enter username: ")
	_, err := fmt.Scanln(&username)
	if err != nil {
		return nil, err
	}

	fmt.Print("Enter password: ")
	bytePwd, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return nil, err
	}

	return []string{username, string(bytePwd)}, nil
}

func (s *State) Register(ctx context.Context) error {
	login, err := s.collectUsernameAndPwd()
	if err != nil {
		return err
	}

	token, err := s.client.Register(ctx, login[0], login[1])
	if err != nil {
		return err
	}

	s.AuthToken = token
	return nil
}

func (s *State) Auth(ctx context.Context) error {
	login, err := s.collectUsernameAndPwd()
	if err != nil {
		return err
	}

	token, err := s.client.Auth(ctx, login[0], login[1])
	if err != nil {
		return err
	}

	s.AuthToken = token
	return nil
}

func (s *State) Upload(ctx context.Context, name string, kind int, data []byte) error {
	err := s.client.Upload(ctx, s.AuthToken, name, kind, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *State) Get(ctx context.Context, name string) (*pamclient.GetResponse, error) {
	data, err := s.client.Get(ctx, s.AuthToken, name)
	if err != nil {
		return nil, err
	}

	return data, err
}

func (s *State) List(ctx context.Context) ([]string, error) {
	names, err := s.client.List(ctx, s.AuthToken)
	if err != nil {
		return names, err
	}

	return names, err
}
