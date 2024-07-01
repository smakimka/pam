package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/smakimka/pam/internal/client/pamclient"
	"github.com/smakimka/pam/internal/client/state"
	"github.com/smakimka/pam/internal/datatypes"
)

type GetCmd struct {
	Name string `arg:"" help:"Name of the data to get"`
}

func (c *GetCmd) Run(ctx context.Context, s *state.State) error {
	data, err := s.Get(ctx, c.Name)
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

	switch data.Kind {
	case datatypes.Text:
		return displayText(c.Name, data)
	default:
		fmt.Println("Unknown data type")
	}

	return nil
}

func displayText(name string, data *pamclient.GetResponse) error {
	fmt.Printf("%s:\n", name)

	fmt.Println(string(data.Data))

	return nil
}
