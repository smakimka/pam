package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/smakimka/pam/internal/client/pamclient"
	"github.com/smakimka/pam/internal/client/state"
)

type AuthCmd struct{}

func (c *AuthCmd) Run(ctx context.Context, state *state.State) error {
	err := state.Auth(ctx)
	if err != nil {
		if errors.Is(err, pamclient.ErrWrongCredentials) {
			fmt.Println("\nWrong username or password")
			return nil
		}
	}

	fmt.Println("\nOk")
	return nil
}
