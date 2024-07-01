package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/smakimka/pam/internal/client/pamclient"
	"github.com/smakimka/pam/internal/client/state"
)

type RegCmd struct{}

func (c *RegCmd) Run(ctx context.Context, state *state.State) error {
	err := state.Register(ctx)
	if err != nil {
		if errors.Is(err, pamclient.ErrUsernameIsTaken) {
			fmt.Println("\nThis username is taken")
			return nil
		}
		return err
	}

	fmt.Println("\nOk")
	return nil
}
