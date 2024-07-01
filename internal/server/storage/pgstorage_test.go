package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
)

type PGStorageTestSuite struct {
	suite.Suite
	storage  *PGStorage
	pool     *dockertest.Pool
	postgres *dockertest.Resource
}

func (s *PGStorageTestSuite) SetupSuite() {
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

	storage, err := NewPGStorage(pgpool)
	if err = storage.Init(context.Background()); err != nil {
		panic(err)
	}

	s.storage = storage
	s.pool = pool
	s.postgres = resource
}

func (s *PGStorageTestSuite) TestGetUser() {
	tests := []struct {
		username string
		wantErr  bool
	}{
		{
			username: "test",
			wantErr:  false,
		},
		{
			username: "1234",
			wantErr:  true,
		},
	}

	_, err := s.storage.p.Exec(context.Background(), `insert into users (username) values ('test')`)
	if err != nil {
		panic(err)
	}

	for _, test := range tests {
		user, err := s.storage.GetUser(context.Background(), test.username)

		if test.wantErr {
			s.Error(err)
		} else {
			s.Equal(test.username, user.Username)
			s.NoError(err)
		}
	}
}

func (s *PGStorageTestSuite) AfterTest(suiteName, testName string) {
	ctx := context.Background()
	err := s.storage.ClearDBAfterTestDoNotUse(ctx)
	if err != nil {
		panic(err)
	}
}

func (s *PGStorageTestSuite) TearDownSuite() {
	if err := s.pool.Purge(s.postgres); err != nil {
		panic(err)
	}
}

func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(PGStorageTestSuite))
}
