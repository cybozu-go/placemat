package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/web"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var podEnterParams struct {
	App string
}

// podEnterCmd represents the podEnter command
var podEnterCmd = &cobra.Command{
	Use:   "enter [--app=APP] POD [COMMAND...]",
	Short: "enter a rkt pod",
	Long:  `enter a rkt pod`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("pod name not specified")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		pod := args[0]
		commands := args[1:]

		uuidCh := make(chan string)
		well.Go(func(ctx context.Context) error {
			var status web.PodStatus
			err := getJSON(ctx, "/pods/"+pod, nil, &status)
			if err != nil {
				return err
			}
			uuid := status.UUID
			uuidCh <- uuid
			return nil
		})

		well.Go(func(ctx context.Context) error {
			var uuid string
			select {
			case uuid = <-uuidCh:
			case <-ctx.Done():
				return ctx.Err()
			}
			args := []string{"enter"}
			if len(podEnterParams.App) > 0 {
				args = append(args, "--app="+podEnterParams.App)
			}
			args = append(args, uuid)
			args = append(args, commands...)
			cmd := exec.CommandContext(ctx, "rkt", args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			return err
		})

		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	podCmd.AddCommand(podEnterCmd)
	podEnterCmd.Flags().StringVar(&podEnterParams.App, "app", "", "name of the app")
}
