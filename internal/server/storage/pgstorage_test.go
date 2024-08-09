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
	storage    *PGStorage
	pgPool     *pgxpool.Pool
	dockerPool *dockertest.Pool
	postgres   *dockertest.Resource
}

func (s *PGStorageTestSuite) SetupSuite() {
	dockerPool, err := dockertest.NewPool("")
	if err != nil {
		panic(err)
	}

	err = dockerPool.Client.Ping()
	if err != nil {
		panic(err)
	}

	resource, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
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
	if err = resource.Expire(60); err != nil {
		panic(err)
	}

	pgpool, err := pgxpool.New(context.Background(), fmt.Sprintf("postgres://postgres:password@localhost:%s/postgres", resource.GetPort("5432/tcp")))
	if err != nil {
		panic(err)
	}

	if err = pgpool.Ping(context.Background()); err != nil {
		panic(err)
	}

	storage, err := NewPGStorage(pgpool)
	if err != nil {
		panic(err)
	}
	if err = storage.Init(context.Background()); err != nil {
		panic(err)
	}

	s.storage = storage
	s.pgPool = pgpool
	s.dockerPool = dockerPool
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

	tx, err := s.pgPool.Begin(ctx)
	if err != nil {
		panic(err)
	}

	_, err = tx.Exec(ctx, `delete from user_data`)
	if err != nil {
		panic(err)
	}
	_, err = tx.Exec(ctx, `delete from auths`)
	if err != nil {
		panic(err)
	}
	_, err = tx.Exec(ctx, `delete from users`)
	if err != nil {
		panic(err)
	}

	if err = tx.Commit(ctx); err != nil {
		panic(err)
	}
}

func (s *PGStorageTestSuite) TearDownSuite() {
	if err := s.dockerPool.Purge(s.postgres); err != nil {
		panic(err)
	}
}

func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(PGStorageTestSuite))
}
