package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/smakimka/pam/internal/client/pamclient"
	"github.com/smakimka/pam/internal/client/state"
)

type ListCmd struct{}

func (c *ListCmd) Run(ctx context.Context, s *state.State) error {
	names, err := s.List(ctx)
	if err != nil {
		if errors.Is(err, pamclient.ErrUnauthenticated) {
			fmt.Println("Please authenticate using the auth command, your token probably expired")
			return nil
		}

		if errors.Is(err, pamclient.ErrDataDoesNotExist) {
			fmt.Println("This data doesn't exist")
			return nil
		}

		return err
	}

	fmt.Println("Your data names:")
	for i, name := range names {
		fmt.Printf("%d. %s\n", i+1, name)
	}

	return nil
}
