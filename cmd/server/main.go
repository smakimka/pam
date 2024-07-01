package main

import (
	"context"
	"flag"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/smakimka/pam/internal/server/config"
	"github.com/smakimka/pam/internal/server/service"
	"github.com/smakimka/pam/internal/server/storage"
)

func main() {
	ctx := context.Background()

	var configPath string
	flag.StringVar(&configPath, "c", "config.json", "path to config file")
	flag.Parse()

	cfg, err := config.New(configPath)
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(ctx, cfg.DBUrl)
	if err != nil {
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

	server := service.NewServer(s)

	fmt.Printf("started server on %s\n", cfg.Addr)
	if err := server.Serve(listen); err != nil {
		panic(err)
	}
}
