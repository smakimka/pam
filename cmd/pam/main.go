package main

import (
	"context"

	"github.com/alecthomas/kong"
	"google.golang.org/grpc"

	"github.com/smakimka/pam/internal/client/certs"
	"github.com/smakimka/pam/internal/client/cli"
	"github.com/smakimka/pam/internal/client/pamclient"
	"github.com/smakimka/pam/internal/client/state"
	"github.com/smakimka/pam/internal/protobuf/pamserver"
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

	tlsCredentials, err := certs.LoadTLSCredentials()
	if err != nil {
		panic(err)
	}

	conn, err := grpc.NewClient(state.ServerAddr, grpc.WithTransportCredentials(tlsCredentials))
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
