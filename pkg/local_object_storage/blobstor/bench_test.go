package blobstor_test

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/blobstor/common"
	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/blobstor/fstree"
	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/blobstor/peapod"
	cid "github.com/nspcc-dev/neofs-sdk-go/container/id"
	"github.com/nspcc-dev/neofs-sdk-go/object"
	oid "github.com/nspcc-dev/neofs-sdk-go/object/id"
	oidtest "github.com/nspcc-dev/neofs-sdk-go/object/id/test"
	"github.com/stretchr/testify/require"
)

func testPeapodPath(tb testing.TB) string {
	return filepath.Join(tb.TempDir(), "peapod.db")
}

func newTestPeapod(tb testing.TB) common.Storage {
	return peapod.New(testPeapodPath(tb), 0o600, 10*time.Millisecond)
}

func newTestFSTree(tb testing.TB) common.Storage {
	return fstree.New(
		fstree.WithDepth(4), // Default.
		fstree.WithPath(tb.TempDir()),
		fstree.WithDirNameLen(1), // Default.
		fstree.WithNoSync(false), // Default.
	)
}

func benchmark(b *testing.B, p common.Storage, objSize uint64, nThreads int) {
	data := make([]byte, objSize)
	rand.Read(data)

	prm := common.PutPrm{
		RawData: data,
	}

	b.ResetTimer()

	for range b.N {
		var wg sync.WaitGroup

		for range nThreads {
			wg.Add(1)
			go func() {
				defer wg.Done()

				prm := prm
				prm.Address = oidtest.Address()

				_, err := p.Put(prm)
				require.NoError(b, err)
			}()
		}

		wg.Wait()
	}
}

func BenchmarkPut(b *testing.B) {
	for _, tc := range []struct {
		objSize  uint64
		nThreads int
	}{
		{1, 1},
		{1, 20},
		{1, 100},
		{1 << 10, 1},
		{1 << 10, 20},
		{1 << 10, 100},
		{100 << 10, 1},
		{100 << 10, 20},
		{100 << 10, 100},
	} {
		b.Run(fmt.Sprintf("size=%d,thread=%d", tc.objSize, tc.nThreads), func(b *testing.B) {
			for name, creat := range map[string]func(testing.TB) common.Storage{
				"peapod": newTestPeapod,
				"fstree": newTestFSTree,
			} {
				b.Run(name, func(b *testing.B) {
					ptt := creat(b)
					require.NoError(b, ptt.Open(false))
					require.NoError(b, ptt.Init())
					b.Cleanup(func() { _ = ptt.Close() })

					benchmark(b, ptt, tc.objSize, tc.nThreads)
				})
			}
		})
	}
}

func BenchmarkGet(b *testing.B) {
	const nObjects = 10000

	for _, tc := range []struct {
		objSize  uint64
		nThreads int
	}{
		{1, 1},
		{1, 20},
		{1, 100},
		{1 << 10, 1},
		{1 << 10, 20},
		{1 << 10, 100},
		{100 << 10, 1},
		{100 << 10, 20},
		{100 << 10, 100},
	} {
		b.Run(fmt.Sprintf("size=%d,thread=%d", tc.objSize, tc.nThreads), func(b *testing.B) {
			for name, creat := range map[string]func(testing.TB) common.Storage{
				"peapod": newTestPeapod,
				"fstree": newTestFSTree,
			} {
				b.Run(name, func(b *testing.B) {
					var objs = make([]oid.Address, 0, nObjects)

					ptt := creat(b)
					require.NoError(b, ptt.Open(false))
					require.NoError(b, ptt.Init())
					b.Cleanup(func() { _ = ptt.Close() })

					obj := object.New()
					data := make([]byte, tc.objSize)
					rand.Read(data)
					obj.SetID(oid.ID{1, 2, 3})
					obj.SetContainerID(cid.ID{1, 2, 3})
					obj.SetPayload(data)

					prm := common.PutPrm{
						RawData: obj.Marshal(),
					}

					var ach = make(chan oid.Address)
					for range 100 {
						go func() {
							for range nObjects / 100 {
								prm := prm

								prm.Address = oidtest.Address()

								_, err := ptt.Put(prm)
								require.NoError(b, err)
								ach <- prm.Address
							}
						}()
					}
					for range nObjects {
						a := <-ach
						objs = append(objs, a)
					}

					b.ResetTimer()
					for n := range b.N {
						var wg sync.WaitGroup

						for i := range tc.nThreads {
							wg.Add(1)
							go func(ind int) {
								defer wg.Done()

								var prm = common.GetPrm{Address: objs[nObjects/tc.nThreads*ind+n%(nObjects/tc.nThreads)]}
								_, err := ptt.Get(prm)
								require.NoError(b, err)
							}(i)
						}

						wg.Wait()
					}
				})
			}
		})
	}
}
