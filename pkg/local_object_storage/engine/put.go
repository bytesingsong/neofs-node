package engine

import (
	"errors"
	"time"

	"github.com/nspcc-dev/neofs-node/pkg/core/object"
	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/blobstor"
	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/blobstor/common"
	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/shard"
	"github.com/nspcc-dev/neofs-node/pkg/util"
	objectSDK "github.com/nspcc-dev/neofs-sdk-go/object"
	oid "github.com/nspcc-dev/neofs-sdk-go/object/id"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

var errPutShard = errors.New("could not put object to any shard")

// Put saves an object to local storage. objBin and hdrLen parameters are
// optional and used to optimize out object marshaling, when used both must
// be valid.
//
// Returns any error encountered that
// did not allow to completely save the object.
//
// Returns an error if executions are blocked (see BlockExecution).
//
// Returns an error of type apistatus.ObjectAlreadyRemoved if the object has been marked as removed.
func (e *StorageEngine) Put(obj *objectSDK.Object, objBin []byte, hdrLen int) error {
	if e.metrics != nil {
		defer elapsed(e.metrics.AddPutDuration)()
	}

	e.blockMtx.RLock()
	defer e.blockMtx.RUnlock()

	if e.blockErr != nil {
		return e.blockErr
	}

	addr := object.AddressOf(obj)

	// In #1146 this check was parallelized, however, it became
	// much slower on fast machines for 4 shards.
	_, err := e.exists(addr)
	if err != nil {
		return err
	}

	var bestShard shardWrapper
	var bestPool util.WorkerPool

	for i, sh := range e.sortedShards(addr) {
		e.mtx.RLock()
		pool, ok := e.shardPools[sh.ID().String()]
		if ok && bestPool == nil {
			bestShard = sh
			bestPool = pool
		}
		e.mtx.RUnlock()
		if !ok {
			// Shard was concurrently removed, skip.
			continue
		}

		putDone, exists, _ := e.putToShard(sh, i, pool, addr, obj, objBin, hdrLen)
		if putDone || exists {
			return nil
		}
	}

	e.log.Debug("failed to put object to shards, trying the best one more",
		zap.Stringer("addr", addr), zap.Stringer("best shard", bestShard.ID()))

	if e.putToShardWithDeadLine(bestShard, 0, bestPool, addr, obj, objBin, hdrLen) {
		return nil
	}

	return errPutShard
}

// putToShard puts object to sh.
// First return value is true iff put has been successfully done.
// Second return value is true iff object already exists.
// Third return value is true iff object cannot be put because of max concurrent load.
func (e *StorageEngine) putToShard(sh shardWrapper, ind int, pool util.WorkerPool, addr oid.Address, obj *objectSDK.Object, objBin []byte, hdrLen int) (bool, bool, bool) {
	var putSuccess, alreadyExists, overloaded bool
	id := sh.ID()

	exitCh := make(chan struct{})

	if err := pool.Submit(func() {
		defer close(exitCh)

		var existPrm shard.ExistsPrm
		existPrm.SetAddress(addr)

		exists, err := sh.Exists(existPrm)
		if err != nil {
			e.log.Warn("object put: check object existence",
				zap.Stringer("addr", addr),
				zap.Stringer("shard", id),
				zap.Error(err))

			if shard.IsErrObjectExpired(err) {
				// object is already found but
				// expired => do nothing with it
				alreadyExists = true
			}

			return // this is not ErrAlreadyRemoved error so we can go to the next shard
		}

		alreadyExists = exists.Exists()
		if alreadyExists {
			if ind != 0 {
				var toMoveItPrm shard.ToMoveItPrm
				toMoveItPrm.SetAddress(addr)

				_, err = sh.ToMoveIt(toMoveItPrm)
				if err != nil {
					e.log.Warn("could not mark object for shard relocation",
						zap.Stringer("shard", id),
						zap.String("error", err.Error()),
					)
				}
			}

			e.log.Debug("object put: object already exists",
				zap.Stringer("shard", id),
				zap.Stringer("addr", addr))

			return
		}

		var putPrm shard.PutPrm
		putPrm.SetObject(obj)
		if objBin != nil {
			putPrm.SetObjectBinary(objBin, hdrLen)
		}

		_, err = sh.Put(putPrm)
		if err != nil {
			if errors.Is(err, shard.ErrReadOnlyMode) || errors.Is(err, blobstor.ErrNoPlaceFound) ||
				errors.Is(err, common.ErrReadOnly) || errors.Is(err, common.ErrNoSpace) {
				e.log.Warn("could not put object to shard",
					zap.Stringer("shard_id", id),
					zap.String("error", err.Error()))
				return
			}

			e.reportShardError(sh, "could not put object to shard", err)
			return
		}

		putSuccess = true
	}); err != nil {
		e.log.Warn("object put: pool task submitting", zap.Stringer("shard", id), zap.Error(err))
		overloaded = errors.Is(err, ants.ErrPoolOverload)
		close(exitCh)
	}

	<-exitCh

	return putSuccess, alreadyExists, overloaded
}

func (e *StorageEngine) putToShardWithDeadLine(sh shardWrapper, ind int, pool util.WorkerPool, addr oid.Address, obj *objectSDK.Object, objBin []byte, hdrLen int) bool {
	var deadline = 30 * time.Second
	timer := time.NewTimer(deadline)
	defer timer.Stop()

	const putCooldown = 100 * time.Millisecond
	ticker := time.NewTicker(putCooldown)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			e.log.Error("could not put object", zap.Stringer("addr", addr), zap.Duration("deadline", deadline))
			return false
		case <-ticker.C:
			putDone, exists, overloaded := e.putToShard(sh, ind, pool, addr, obj, objBin, hdrLen)
			if overloaded {
				ticker.Reset(putCooldown)
				continue
			}

			return putDone || exists
		}
	}
}
