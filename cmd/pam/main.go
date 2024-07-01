package main

import (
	"context"

	"github.com/alecthomas/kong"
	"github.com/smakimka/pam/internal/client/cli"
	"github.com/smakimka/pam/internal/client/pamclient"
	"github.com/smakimka/pam/internal/client/state"
	"github.com/smakimka/pam/internal/protobuf/pamserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx := context.Background()

	state, err := state.Open()
	defer state.Close()
	if err != nil {
		panic(err)
	}

	if state.ServerAddr == "" {
		if err = state.ReadServerAddr(); err != nil {
			panic(err)
		}
	}

	conn, err := grpc.NewClient(state.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	grpcClient := pamserver.NewPamServerClient(conn)

	client := pamclient.NewGRPCClient(grpcClient)
	state.SetClient(client)

	context := kong.Parse(&cli.CLI, kong.BindTo(ctx, (*context.Context)(nil)))

	err = context.Run(ctx, state)
	context.FatalIfErrorf(err)
}
