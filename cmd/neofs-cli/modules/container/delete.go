package container

import (
	"fmt"
	"time"

	internalclient "github.com/nspcc-dev/neofs-node/cmd/neofs-cli/internal/client"
	"github.com/nspcc-dev/neofs-node/cmd/neofs-cli/internal/common"
	"github.com/nspcc-dev/neofs-node/cmd/neofs-cli/internal/commonflags"
	"github.com/nspcc-dev/neofs-node/cmd/neofs-cli/internal/key"
	"github.com/nspcc-dev/neofs-sdk-go/client"
	neofsecdsa "github.com/nspcc-dev/neofs-sdk-go/crypto/ecdsa"
	objectSDK "github.com/nspcc-dev/neofs-sdk-go/object"
	"github.com/nspcc-dev/neofs-sdk-go/user"
	"github.com/spf13/cobra"
)

var deleteContainerCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete existing container",
	Long: `Delete existing container.
Only owner of the container has a permission to remove container.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		id, err := parseContainerID()
		if err != nil {
			return err
		}

		tok, err := getSession(cmd)
		if err != nil {
			return err
		}

		pk, err := key.Get(cmd)
		if err != nil {
			return err
		}

		ctx, cancel := getAwaitContext(cmd)
		defer cancel()

		cli, err := internalclient.GetSDKClientByFlag(ctx, commonflags.RPC)
		if err != nil {
			return err
		}
		defer cli.Close()

		if force, _ := cmd.Flags().GetBool(commonflags.ForceFlag); !force {
			common.PrintVerbose(cmd, "Reading the container to check ownership...")

			var getPrm internalclient.GetContainerPrm
			getPrm.SetClient(cli)
			getPrm.SetContainer(id)

			resGet, err := internalclient.GetContainer(ctx, getPrm)
			if err != nil {
				return fmt.Errorf("can't get the container: %w", err)
			}

			owner := resGet.Container().Owner()

			if tok != nil {
				common.PrintVerbose(cmd, "Checking session issuer...")

				if tok.Issuer() != owner {
					return fmt.Errorf("session issuer differs with the container owner: expected %s, has %s", owner, tok.Issuer())
				}
			} else {
				common.PrintVerbose(cmd, "Checking provided account...")

				acc := user.NewFromECDSAPublicKey(pk.PublicKey)

				if acc != owner {
					return fmt.Errorf("provided account differs with the container owner: expected %s, has %s", owner, acc)
				}
			}

			common.PrintVerbose(cmd, "Account matches the container owner.")

			if tok != nil {
				common.PrintVerbose(cmd, "Skip searching for LOCK objects - session provided.")
			} else {
				fs := objectSDK.NewSearchFilters()
				fs.AddTypeFilter(objectSDK.MatchStringEqual, objectSDK.TypeLock)

				var opts client.SearchObjectsOptions
				opts.SetCount(1)

				common.PrintVerbose(cmd, "Searching for LOCK objects...")

				res, _, err := cli.SearchObjects(ctx, id, fs, nil, "", (*neofsecdsa.Signer)(pk), opts)
				if err != nil {
					return fmt.Errorf("can't search for LOCK objects: %w", err)
				}

				if len(res) != 0 {
					return fmt.Errorf("Container wasn't removed because LOCK objects were found.\n"+
						"Use --%s flag to remove anyway.", commonflags.ForceFlag)
				}
			}
		}

		var delPrm internalclient.DeleteContainerPrm
		delPrm.SetClient(cli)
		delPrm.SetContainer(id)
		delPrm.SetPrivateKey(*pk)

		if tok != nil {
			delPrm.WithinSession(*tok)
		}

		_, err = internalclient.DeleteContainer(ctx, delPrm)
		if err != nil {
			return fmt.Errorf("rpc error: %w", err)
		}

		cmd.Println("container removal request accepted for processing (the operation may not be completed yet)")

		if containerAwait {
			cmd.Println("awaiting...")

			var getPrm internalclient.GetContainerPrm
			getPrm.SetClient(cli)
			getPrm.SetContainer(id)

			const waitInterval = time.Second

			t := time.NewTimer(waitInterval)

			for ; ; t.Reset(waitInterval) {
				select {
				case <-ctx.Done():
					return fmt.Errorf("container deletion: %w", common.ErrAwaitTimeout)
				case <-t.C:
				}

				_, err := internalclient.GetContainer(ctx, getPrm)
				if err != nil {
					cmd.Println("container has been removed:", containerID)
					return nil
				}
			}
		}
		return nil
	},
}

func initContainerDeleteCmd() {
	flags := deleteContainerCmd.Flags()

	flags.StringP(commonflags.WalletPath, commonflags.WalletPathShorthand, commonflags.WalletPathDefault, commonflags.WalletPathUsage)
	flags.StringP(commonflags.Account, commonflags.AccountShorthand, commonflags.AccountDefault, commonflags.AccountUsage)
	flags.StringP(commonflags.RPC, commonflags.RPCShorthand, commonflags.RPCDefault, commonflags.RPCUsage)

	flags.StringVar(&containerID, commonflags.CIDFlag, "", commonflags.CIDFlagUsage)
	flags.BoolVar(&containerAwait, "await", false, fmt.Sprintf("Block execution until container is removed. "+
		"Increases default execution timeout to %.0fs", awaitTimeout.Seconds())) // simple %s notation prints 1m0s https://github.com/golang/go/issues/39064
	flags.BoolP(commonflags.ForceFlag, commonflags.ForceFlagShorthand, false, "Skip validation checks (ownership, presence of LOCK objects)")
}
