package control

import (
	"fmt"
	"strings"

	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/util"
	rawclient "github.com/nspcc-dev/neofs-api-go/v2/rpc/client"
	"github.com/nspcc-dev/neofs-node/cmd/neofs-cli/internal/commonflags"
	"github.com/nspcc-dev/neofs-node/cmd/neofs-cli/internal/key"
	ircontrol "github.com/nspcc-dev/neofs-node/pkg/services/control/ir"
	ircontrolsrv "github.com/nspcc-dev/neofs-node/pkg/services/control/ir/server"
	"github.com/spf13/cobra"
)

const notaryMethodFlag = "method"

var notaryRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Create and send a notary request",
	Long: "Create and send a notary request with one of the following methods:\n" +
		"- newEpoch, transaction for creating of new NeoFS epoch event in FS chain, no args\n" +
		"- setConfig, transaction to add/update global config value in the NeoFS network, 1 arg in the form key=val\n" +
		"- removeNode, transaction to move nodes to the Offline state in the candidates list, 1 arg is the public key of the node",
	RunE: notaryRequest,
}

func initControlNotaryRequestCmd() {
	initControlFlags(notaryRequestCmd)

	flags := notaryRequestCmd.Flags()
	flags.String(notaryMethodFlag, "", "Requested method")
}

func notaryRequest(cmd *cobra.Command, args []string) error {
	ctx, cancel := commonflags.GetCommandContext(cmd)
	defer cancel()

	pk, err := key.Get(cmd)
	if err != nil {
		return err
	}

	cli, err := getClient(ctx)
	if err != nil {
		return err
	}

	req := new(ircontrol.NotaryRequestRequest)
	body := new(ircontrol.NotaryRequestRequest_Body)
	req.SetBody(body)

	method, _ := cmd.Flags().GetString(notaryMethodFlag)
	body.SetMethod(method)

	switch method {
	case "newEpoch":
		if len(args) > 0 {
			cmd.Println("method 'newEpoch', but the args provided, they will be ignored")
		}
	case "setConfig":
		if len(args) != 1 {
			return fmt.Errorf("invalid number of args provided for 'setConfig', expected 1, got %d", len(args))
		}

		kv := strings.SplitN(args[0], "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid parameter format: must be 'key=val', got: %s", args[0])
		}

		body.SetArgs([][]byte{[]byte(kv[0]), []byte(kv[1])})
	case "removeNode":
		if len(args) != 1 {
			return fmt.Errorf("invalid number of args provided for 'removeNode', expected 1, got %d", len(args))
		}
		key, err := keys.NewPublicKeyFromString(args[0])
		if err != nil {
			return err
		}

		body.SetArgs([][]byte{key.Bytes()})
	}

	err = ircontrolsrv.SignMessage(pk, req)
	if err != nil {
		return fmt.Errorf("could not sign request: %w", err)
	}

	var resp *ircontrol.NotaryRequestResponse
	err = cli.ExecRaw(func(client *rawclient.Client) error {
		resp, err = ircontrol.NotaryRequest(client, req)
		return err
	})
	if err != nil {
		return fmt.Errorf("rpc error: %w", err)
	}

	err = verifyResponse(resp.GetSignature(), resp.GetBody())
	if err != nil {
		return err
	}

	hashB := resp.GetBody().GetHash()

	hash, err := util.Uint256DecodeBytesBE(hashB)
	if err != nil {
		return fmt.Errorf("failed to decode hash %v: %w", hashB, err)
	}
	cmd.Printf("Transaction Hash: %s\n", hash.String())
	return nil
}
