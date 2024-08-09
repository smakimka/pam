package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/smakimka/pam/internal/server/certs"
	"github.com/smakimka/pam/internal/server/config"
	"github.com/smakimka/pam/internal/server/service"
	"github.com/smakimka/pam/internal/server/storage"
)

func main() {
	ctx := context.Background()

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(ctx, cfg.DBUrl)
	if err != nil {
		panic(err)
	}

	if err = waitForPostgres(ctx, pool); err != nil {
		panic(err)
	}

	s, err := storage.NewPGStorage(pool)
	if err != nil {
		panic(err)
	}
	err = s.Init(ctx)
	if err != nil {
		panic(err)
	}

	listen, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		panic(err)
	}

	tlsCredentials, err := certs.LoadTLSCredentials()
	if err != nil {
		panic(err)
	}
	server := service.NewServer(s, tlsCredentials, cfg.AuthTokenExpiryTimeSec)

	fmt.Printf("started server on %s\n", cfg.Addr)
	if err := server.Serve(listen); err != nil {
		panic(err)
	}
}

func waitForPostgres(ctx context.Context, p *pgxpool.Pool) error {
	timeoutTimer := time.NewTimer(10 * time.Second)

	okChan := make(chan struct{})
	go func() {
		for {
			err := p.Ping(ctx)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			okChan <- struct{}{}
			break
		}
	}()

	select {
	case <-timeoutTimer.C:
		return errors.New("Couldn't reach postgres'")
	case <-okChan:
		return nil
	}
}
