package innerring

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neofs-node/pkg/core/container"
	"github.com/nspcc-dev/neofs-node/pkg/innerring/processors/settlement/basic"
	"github.com/nspcc-dev/neofs-node/pkg/innerring/processors/settlement/common"
	balanceClient "github.com/nspcc-dev/neofs-node/pkg/morph/client/balance"
	containerClient "github.com/nspcc-dev/neofs-node/pkg/morph/client/container"
	netmapClient "github.com/nspcc-dev/neofs-node/pkg/morph/client/netmap"
	containerAPI "github.com/nspcc-dev/neofs-sdk-go/container"
	cid "github.com/nspcc-dev/neofs-sdk-go/container/id"
	netmapAPI "github.com/nspcc-dev/neofs-sdk-go/netmap"
	"github.com/nspcc-dev/neofs-sdk-go/user"
	"go.uber.org/zap"
)

const (
	basicIncomeSettlementContext = "basic income"
)

type settlementDeps struct {
	log *zap.Logger

	cnrSrc container.Source

	nmClient *netmapClient.Client

	balanceClient *balanceClient.Client

	settlementCtx string
}

type basicIncomeSettlementDeps struct {
	settlementDeps
	cnrClient *containerClient.Client
}

type basicSettlementConstructor struct {
	dep *basicIncomeSettlementDeps
}

type containerWrapper containerAPI.Container

type nodeInfoWrapper struct {
	ni netmapAPI.NodeInfo
}

func (n nodeInfoWrapper) PublicKey() []byte {
	return n.ni.PublicKey()
}

func (n nodeInfoWrapper) Price() *big.Int {
	return big.NewInt(int64(n.ni.Price()))
}

func (c containerWrapper) Owner() user.ID {
	return (containerAPI.Container)(c).Owner()
}

func (s settlementDeps) ContainerInfo(cid cid.ID) (common.ContainerInfo, error) {
	cnr, err := s.cnrSrc.Get(cid)
	if err != nil {
		return nil, fmt.Errorf("could not get container from storage: %w", err)
	}

	return (containerWrapper)(cnr), nil
}

func (s settlementDeps) buildContainer(e uint64, cid cid.ID) ([][]netmapAPI.NodeInfo, *netmapAPI.NetMap, error) {
	var (
		nm  *netmapAPI.NetMap
		err error
	)

	if e > 0 {
		nm, err = s.nmClient.GetNetMapByEpoch(e)
	} else {
		nm, err = s.nmClient.NetMap()
	}

	if err != nil {
		return nil, nil, fmt.Errorf("could not get network map from storage: %w", err)
	}

	cnr, err := s.cnrSrc.Get(cid)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get container from FS chain: %w", err)
	}

	cn, err := nm.ContainerNodes(
		cnr.PlacementPolicy(),
		cid,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not calculate container nodes: %w", err)
	}

	return cn, nm, nil
}

func (s settlementDeps) ContainerNodes(e uint64, cid cid.ID) ([]common.NodeInfo, error) {
	cn, _, err := s.buildContainer(e, cid)
	if err != nil {
		return nil, err
	}

	var sz int

	for i := range cn {
		sz += len(cn[i])
	}

	res := make([]common.NodeInfo, 0, sz)

	for i := range cn {
		for j := range cn[i] {
			if cn[i][j].IsOnline() {
				res = append(res, nodeInfoWrapper{
					ni: cn[i][j],
				})
			}
		}
	}

	return res, nil
}

func (s settlementDeps) ResolveKey(ni common.NodeInfo) (*user.ID, error) {
	pubKey, err := keys.NewPublicKeyFromBytes(ni.PublicKey(), elliptic.P256())
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}

	id := user.NewFromECDSAPublicKey(ecdsa.PublicKey(*pubKey))

	return &id, nil
}

func (s settlementDeps) Transfer(sender, recipient user.ID, amount *big.Int, details []byte) {
	if s.settlementCtx == "" {
		panic("unknown settlement deps context")
	}

	log := s.log.With(
		zap.Stringer("sender", sender),
		zap.Stringer("recipient", recipient),
		zap.Stringer("amount (GASe-12)", amount),
		zap.String("details", hex.EncodeToString(details)),
	)

	err := s.balanceClient.TransferX(sender, recipient, amount, details)
	if err != nil {
		log.Error(fmt.Sprintf("%s: could not send transfer", s.settlementCtx),
			zap.Error(err),
		)

		return
	}

	log.Debug(fmt.Sprintf("%s: transfer was successfully sent", s.settlementCtx))
}

func (b basicIncomeSettlementDeps) BasicRate() (uint64, error) {
	return b.nmClient.BasicIncomeRate()
}

func (b basicIncomeSettlementDeps) Estimations(epoch uint64) (map[cid.ID]*containerClient.Estimations, error) {
	return b.cnrClient.ListLoadEstimationsByEpoch(epoch)
}

func (b basicIncomeSettlementDeps) Balance(id user.ID) (*big.Int, error) {
	return b.balanceClient.BalanceOf(id)
}

func (b *basicSettlementConstructor) CreateContext(epoch uint64) (*basic.IncomeSettlementContext, error) {
	return basic.NewIncomeSettlementContext(&basic.IncomeSettlementContextPrms{
		Log:         b.dep.log,
		Epoch:       epoch,
		Rate:        b.dep,
		Estimations: b.dep,
		Balances:    b.dep,
		Container:   b.dep,
		Placement:   b.dep,
		Exchange:    b.dep,
		Accounts:    b.dep,
	}), nil
}
