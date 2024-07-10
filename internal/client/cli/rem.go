package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/smakimka/pam/internal/client/pamclient"
	"github.com/smakimka/pam/internal/client/state"
	"github.com/smakimka/pam/internal/datatypes"
)

type RemCmd struct {
	DataType string `arg:"" help:"what type of data to remember.Options (text)"`
}

func (c *RemCmd) Run(ctx context.Context, s *state.State) error {
	switch c.DataType {
	case "text":
		return rememberText(ctx, s)
	default:
		fmt.Println("no such data type, available are: text")
	}
	return nil
}

func rememberText(ctx context.Context, s *state.State) error {
	var name string
	fmt.Print("Enter name: ")
	_, err := fmt.Scanln(&name)
	if err != nil {
		return err
	}

	var text string
	fmt.Print("Enter text: ")
	_, err = fmt.Scanln(&text)
	if err != nil {
		return err
	}

	if err = s.Upload(ctx, name, datatypes.Text, []byte(text)); err != nil {
		if errors.Is(err, pamclient.ErrUnauthenticated) {
			fmt.Println("Please authenticate using the auth command, your token probably expired")
			return nil
		}
		return err
	}

	fmt.Println("Ok")
	return nil
}
