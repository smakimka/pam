package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/smakimka/pam/internal/datatypes"
	"github.com/smakimka/pam/internal/protobuf/pamserver"
	"github.com/smakimka/pam/internal/server/model"
	"github.com/smakimka/pam/internal/server/storage"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type ServiceTestSuite struct {
	suite.Suite
	storage  *storage.PGStorage
	pool     *dockertest.Pool
	postgres *dockertest.Resource
}

func (s *ServiceTestSuite) SetupSuite() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(err)
	}

	err = pool.Client.Ping()
	if err != nil {
		panic(err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16.3",
		Env:        []string{"POSTGRES_PASSWORD=password"}},
		func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second * 5)
	resource.Expire(60)

	pgpool, err := pgxpool.New(context.Background(), fmt.Sprintf("postgres://postgres:password@localhost:%s/postgres", resource.GetPort("5432/tcp")))
	if err != nil {
		panic(err)
	}

	if err = pgpool.Ping(context.Background()); err != nil {
		panic(err)
	}

	storage, err := storage.NewPGStorage(pgpool)
	if err = storage.Init(context.Background()); err != nil {
		panic(err)
	}

	s.storage = storage
	s.pool = pool
	s.postgres = resource
}

func (s *ServiceTestSuite) TestRegister() {
	tests := []struct {
		in        pamserver.AuthData
		wantErr   bool
		wantToken bool
	}{
		{
			in:        pamserver.AuthData{Username: "new_user", Pwd: "pwd"},
			wantErr:   false,
			wantToken: true,
		},
		{
			in:        pamserver.AuthData{Username: "new_user", Pwd: "pwd"},
			wantErr:   true,
			wantToken: false,
		},
	}
	ctx := context.Background()
	service := newPamSerice(s.storage, 300)

	for _, test := range tests {
		out, err := service.Register(ctx, &test.in)
		s.NoError(err)

		if test.wantErr {
			s.NotEqual("", out.Error)
		} else {
			s.Equal("", out.Error)
		}

		if test.wantToken {
			s.NotEqual("", out.Token)
		} else {
			s.Equal("", out.Token)
		}
	}
}

func (s *ServiceTestSuite) TestAuthenticate() {
	tests := []struct {
		in         pamserver.AuthData
		wantErr    bool
		wantToken  bool
		wantBigErr bool
	}{
		{
			in:         pamserver.AuthData{Username: "user", Pwd: "pwd"},
			wantErr:    false,
			wantToken:  true,
			wantBigErr: false,
		},
		{
			in:         pamserver.AuthData{Username: "new_user", Pwd: "pwd"},
			wantErr:    false,
			wantToken:  false,
			wantBigErr: true,
		},
	}
	ctx := context.Background()
	service := newPamSerice(s.storage, 300)

	pwd, err := bcrypt.GenerateFromPassword([]byte("pwd"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	_, err = s.storage.CreateUser(ctx, "user", pwd)
	if err != nil {
		panic(err)
	}

	for _, test := range tests {
		out, err := service.Authenticate(ctx, &test.in)
		if test.wantBigErr {
			s.Error(err)
		} else {
			s.NoError(err)
		}

		if test.wantErr {
			s.NotEqual("", out.Error)
		} else {
			s.Equal("", out.Error)
		}

		if test.wantToken {
			s.NotEqual("", out.Token)
		} else {
			s.Equal("", out.Token)
		}

	}
}

func (s *ServiceTestSuite) TestUpload() {
	tests := []struct {
		in            pamserver.UploadData
		wantOut       pamserver.UploadResponse
		wantErr       bool
		wantAllNames  []string
		wantDataName  string
		wantDataValue []byte
	}{
		{
			in:            pamserver.UploadData{Name: "test_data", Type: int32(datatypes.Text), Data: []byte("123")},
			wantOut:       pamserver.UploadResponse{},
			wantErr:       false,
			wantAllNames:  []string{"test_data"},
			wantDataName:  "test_data",
			wantDataValue: []byte("123"),
		},
		{
			in:            pamserver.UploadData{Name: "test_data", Type: int32(datatypes.Text), Data: []byte("1234")},
			wantOut:       pamserver.UploadResponse{},
			wantErr:       false,
			wantAllNames:  []string{"test_data"},
			wantDataName:  "test_data",
			wantDataValue: []byte("1234"),
		},
		{
			in:            pamserver.UploadData{Name: "test_data2", Type: int32(datatypes.Text), Data: []byte("222")},
			wantOut:       pamserver.UploadResponse{},
			wantErr:       false,
			wantAllNames:  []string{"test_data", "test_data2"},
			wantDataName:  "test_data2",
			wantDataValue: []byte("222"),
		},
	}
	ctx := context.Background()
	service := newPamSerice(s.storage, 300)

	userID, err := s.storage.CreateUser(ctx, "test_user", []byte("123"))
	if err != nil {
		panic(err)
	}

	_, err = s.storage.CreateAuthToken(ctx, userID, "token", time.Now().Add(60*time.Second))
	if err != nil {
		panic(err)
	}

	ctx = context.WithValue(ctx, model.UserID, userID)
	ctx = context.WithValue(ctx, model.AuthToken, "token")

	for _, test := range tests {
		out, err := service.Upload(ctx, &test.in)
		if test.wantErr {
			s.Error(err)
			continue
		}

		s.NoError(err)
		s.Equal(test.wantOut.Error, out.Error)

		data, err := s.storage.GetData(ctx, userID, test.wantDataName)
		s.NoError(err)
		s.Equal(test.wantDataValue, data.Bytes)

		dataNames, err := s.storage.GetDataNames(ctx, userID)
		s.NoError(err)
		s.EqualValues(test.wantAllNames, dataNames)
	}
}

func (s *ServiceTestSuite) GetData() {
	tests := []struct {
		in            pamserver.GetData
		wantOut       pamserver.GetDataResponse
		wantErr       bool
		wantDataName  string
		wantDataValue []byte
	}{
		{
			in:            pamserver.GetData{Name: "test_data"},
			wantOut:       pamserver.GetDataResponse{Kind: int32(datatypes.Text), Data: []byte("test")},
			wantErr:       false,
			wantDataName:  "test_data",
			wantDataValue: []byte("test"),
		},
		{
			in:            pamserver.GetData{Name: "not_test_data"},
			wantOut:       pamserver.GetDataResponse{},
			wantErr:       true,
			wantDataName:  "test_data",
			wantDataValue: []byte("1234"),
		},
	}
	ctx := context.Background()
	service := newPamSerice(s.storage, 300)

	userID, err := s.storage.CreateUser(ctx, "test_user", []byte("123"))
	if err != nil {
		panic(err)
	}

	_, err = s.storage.CreateAuthToken(ctx, userID, "token", time.Now().Add(60*time.Second))
	if err != nil {
		panic(err)
	}

	ctx = context.WithValue(ctx, model.UserID, userID)
	ctx = context.WithValue(ctx, model.AuthToken, "token")

	_, err = s.storage.UpsertData(ctx, userID, "test_data", datatypes.Text, []byte("test"))
	if err != nil {
		panic(err)
	}

	for _, test := range tests {
		out, err := service.Get(ctx, &test.in)
		if test.wantErr {
			s.Error(err)
			continue
		}

		s.NoError(err)
		s.Equal(test.wantOut.Kind, out.Kind)
		s.Equal(test.wantOut.Data, out.Data)
	}
}

func (s *ServiceTestSuite) GetDataNames() {
	tests := []struct {
		in            pamserver.GetDataNames
		wantOut       pamserver.GetDataNamesResponse
		wantErr       bool
		wantDataName  string
		wantDataValue []byte
	}{
		{
			in:            pamserver.GetDataNames{},
			wantOut:       pamserver.GetDataNamesResponse{Names: []string{"test_data"}},
			wantErr:       false,
			wantDataName:  "test_data",
			wantDataValue: []byte("test"),
		},
	}
	ctx := context.Background()
	service := newPamSerice(s.storage, 300)

	userID, err := s.storage.CreateUser(ctx, "test_user", []byte("123"))
	if err != nil {
		panic(err)
	}

	_, err = s.storage.CreateAuthToken(ctx, userID, "token", time.Now().Add(60*time.Second))
	if err != nil {
		panic(err)
	}

	ctx = context.WithValue(ctx, model.UserID, userID)
	ctx = context.WithValue(ctx, model.AuthToken, "token")

	_, err = s.storage.UpsertData(ctx, userID, "test_data", datatypes.Text, []byte("test"))
	if err != nil {
		panic(err)
	}

	for _, test := range tests {
		out, err := service.GetNames(ctx, &test.in)
		if test.wantErr {
			s.Error(err)
			continue
		}

		s.NoError(err)
		s.EqualValues(test.wantOut.Names, out.Names)
	}
}

func (s *ServiceTestSuite) AfterTest(suiteName, testName string) {
	ctx := context.Background()
	err := s.storage.ClearDBAfterTestDoNotUse(ctx)
	if err != nil {
		panic(err)
	}
}

func (s *ServiceTestSuite) TearDownSuite() {
	if err := s.pool.Purge(s.postgres); err != nil {
		panic(err)
	}
}

func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}
