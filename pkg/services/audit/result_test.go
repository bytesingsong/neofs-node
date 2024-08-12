package audit_test

import (
	"math/rand"
	"testing"

	apiaudit "github.com/nspcc-dev/neofs-api-go/v2/audit"
	"github.com/nspcc-dev/neofs-api-go/v2/refs"
	"github.com/nspcc-dev/neofs-node/pkg/services/audit"
	cidtest "github.com/nspcc-dev/neofs-sdk-go/container/id/test"
	oid "github.com/nspcc-dev/neofs-sdk-go/object/id"
	oidtest "github.com/nspcc-dev/neofs-sdk-go/object/id/test"
	"github.com/stretchr/testify/require"
)

func anyValidAuditResult() audit.Result {
	return audit.NewResult([]byte("any_public_key"), rand.Uint64(), cidtest.ID())
}

func TestResultProtocolVersion(t *testing.T) {
	r := anyValidAuditResult()
	var msg apiaudit.DataAuditResult

	require.NoError(t, msg.Unmarshal(r.Marshal()))
	ver := msg.GetVersion()
	require.EqualValues(t, 2, ver.GetMajor())
	require.EqualValues(t, 16, ver.GetMinor())

	ver.SetMajor(100)
	ver.SetMinor(500)
	msg.SetVersion(ver)
	require.NoError(t, r.Unmarshal(msg.StableMarshal(nil)))
	var msg2 apiaudit.DataAuditResult
	require.NoError(t, msg2.Unmarshal(r.Marshal()))
	ver = msg.GetVersion()
	require.EqualValues(t, 100, ver.GetMajor())
	require.EqualValues(t, 500, ver.GetMinor())
}

func TestResultMarshaling(t *testing.T) {
	t.Run("audit epoch", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.AuditEpoch = 0
		dst.AuditEpoch = 1
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Zero(t, dst.AuditEpoch)

		src.AuditEpoch = rand.Uint64()
		if src.AuditEpoch == dst.AuditEpoch {
			src.AuditEpoch++
		}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.AuditEpoch, dst.AuditEpoch)
	})
	t.Run("container", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.Container = cidtest.ID()
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.Container, dst.Container)

		for {
			cnr := cidtest.ID()
			if cnr != src.Container {
				src.Container = cnr
				break
			}
		}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.Container, dst.Container)
	})
	t.Run("auditor key", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.AuditorPublicKey = nil
		dst.AuditorPublicKey = []byte("some_key")
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Nil(t, dst.AuditorPublicKey)

		src.AuditorPublicKey = make([]byte, 33)
		rand.Read(src.AuditorPublicKey)
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.AuditorPublicKey, dst.AuditorPublicKey)
	})
	t.Run("completed", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.Completed = false
		dst.Completed = true
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.False(t, dst.Completed)

		src.Completed = true
		dst.Completed = false
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.True(t, dst.Completed)
	})
	t.Run("PoP hits", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PoP.Hits = 0
		dst.PoP.Hits = 1
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Zero(t, dst.PoP.Hits)

		src.PoP.Hits = rand.Uint32()
		if src.PoP.Hits == dst.PoP.Hits {
			src.PoP.Hits++
		}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PoP.Hits, dst.PoP.Hits)
	})
	t.Run("PoP misses", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PoP.Misses = 0
		dst.PoP.Misses = 1
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Zero(t, dst.PoP.Misses)

		src.PoP.Misses = rand.Uint32()
		if src.PoP.Misses == dst.PoP.Misses {
			src.PoP.Misses++
		}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PoP.Misses, dst.PoP.Misses)
	})
	t.Run("PoP failures", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PoP.Failures = 0
		dst.PoP.Failures = 1
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Zero(t, dst.PoP.Failures)

		src.PoP.Failures = rand.Uint32()
		if src.PoP.Failures == dst.PoP.Failures {
			src.PoP.Failures++
		}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PoP.Failures, dst.PoP.Failures)
	})
	t.Run("PoR requests", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PoR.Requests = 0
		dst.PoR.Requests = 1
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Zero(t, dst.PoR.Requests)

		src.PoR.Requests = rand.Uint32()
		if src.PoR.Requests == dst.PoR.Requests {
			src.PoR.Requests++
		}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PoR.Requests, dst.PoR.Requests)
	})
	t.Run("PoR retries", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PoR.Retries = 0
		dst.PoR.Retries = 1
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Zero(t, dst.PoR.Retries)

		src.PoR.Retries = rand.Uint32()
		if src.PoR.Retries == dst.PoR.Retries {
			src.PoR.Retries++
		}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PoR.Retries, dst.PoR.Retries)
	})
	t.Run("PoR passed SG", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PoR.PassedStorageGroups = nil
		dst.PoR.PassedStorageGroups = []oid.ID{oidtest.ID(), oidtest.ID()}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Nil(t, dst.PoR.PassedStorageGroups)

		src.PoR.PassedStorageGroups = []oid.ID{oidtest.ID(), oidtest.ID(), oidtest.ID()}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PoR.PassedStorageGroups, dst.PoR.PassedStorageGroups)
	})
	t.Run("PoR failed SG", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PoR.FailedStorageGroups = nil
		dst.PoR.FailedStorageGroups = []oid.ID{oidtest.ID(), oidtest.ID()}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Nil(t, dst.PoR.FailedStorageGroups)

		src.PoR.FailedStorageGroups = []oid.ID{oidtest.ID(), oidtest.ID(), oidtest.ID()}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PoR.FailedStorageGroups, dst.PoR.FailedStorageGroups)
	})
	t.Run("PDP passed nodes", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PDP.PassedStorageNodes = nil
		dst.PDP.PassedStorageNodes = [][]byte{[]byte("any_key1"), []byte("any_key2")}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Nil(t, dst.PDP.PassedStorageNodes)

		src.PDP.PassedStorageNodes = [][]byte{[]byte("any_key1"), []byte("any_key2"), []byte("any_key3")}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PDP.PassedStorageNodes, dst.PDP.PassedStorageNodes)
	})
	t.Run("PDP failed nodes", func(t *testing.T) {
		src := anyValidAuditResult()
		var dst audit.Result

		src.PDP.FailedStorageNodes = nil
		dst.PDP.FailedStorageNodes = [][]byte{[]byte("any_key1"), []byte("any_key2")}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Nil(t, dst.PDP.FailedStorageNodes)

		src.PDP.FailedStorageNodes = [][]byte{[]byte("any_key1"), []byte("any_key2"), []byte("any_key3")}
		require.NoError(t, dst.Unmarshal(src.Marshal()))
		require.Equal(t, src.PDP.FailedStorageNodes, dst.PDP.FailedStorageNodes)
	})
}

func TestResultUnmarshalingFailures(t *testing.T) {
	t.Run("invalid protobuf", func(t *testing.T) {
		var r audit.Result
		require.ErrorContains(t, r.Unmarshal([]byte("definitely_not_protobuf")), "decode protobuf")
	})
	t.Run("invalid fields", func(t *testing.T) {
		for _, testCase := range []struct {
			name    string
			err     string
			corrupt func(*apiaudit.DataAuditResult)
		}{
			{name: "missing container", err: "missing container", corrupt: func(r *apiaudit.DataAuditResult) {
				r.SetContainerID(nil)
			}},
			{name: "invalid container/nil value", err: "invalid container: invalid length 0", corrupt: func(r *apiaudit.DataAuditResult) {
				r.SetContainerID(new(refs.ContainerID))
			}},
			{name: "invalid container/empty value", err: "invalid container: invalid length 0", corrupt: func(r *apiaudit.DataAuditResult) {
				var id refs.ContainerID
				id.SetValue([]byte{})
				r.SetContainerID(&id)
			}},
			{name: "invalid container/wrong length", err: "invalid container: invalid length 31", corrupt: func(r *apiaudit.DataAuditResult) {
				var id refs.ContainerID
				id.SetValue(make([]byte, 31))
				r.SetContainerID(&id)
			}},
			{name: "invalid passed SG/nil value", err: "invalid passed storage group #1: invalid length 0", corrupt: func(r *apiaudit.DataAuditResult) {
				ids := make([]refs.ObjectID, 3)
				ids[0].SetValue(randomObjectID())
				ids[2].SetValue(randomObjectID())
				r.SetPassSG(ids)
			}},
			{name: "invalid passed SG/empty value", err: "invalid passed storage group #1: invalid length 0", corrupt: func(r *apiaudit.DataAuditResult) {
				ids := make([]refs.ObjectID, 3)
				ids[0].SetValue(randomObjectID())
				ids[1].SetValue([]byte{})
				ids[2].SetValue(randomObjectID())
				r.SetPassSG(ids)
			}},
			{name: "invalid passed SG/wrong length", err: "invalid passed storage group #1: invalid length 31", corrupt: func(r *apiaudit.DataAuditResult) {
				ids := make([]refs.ObjectID, 3)
				ids[0].SetValue(randomObjectID())
				ids[1].SetValue(make([]byte, 31))
				ids[2].SetValue(randomObjectID())
				r.SetPassSG(ids)
			}},
			{name: "invalid failed SG/nil value", err: "invalid failed storage group #1: invalid length 0", corrupt: func(r *apiaudit.DataAuditResult) {
				ids := make([]refs.ObjectID, 3)
				ids[0].SetValue(randomObjectID())
				ids[2].SetValue(randomObjectID())
				r.SetFailSG(ids)
			}},
			{name: "invalid failed SG/empty value", err: "invalid failed storage group #1: invalid length 0", corrupt: func(r *apiaudit.DataAuditResult) {
				ids := make([]refs.ObjectID, 3)
				ids[0].SetValue(randomObjectID())
				ids[1].SetValue([]byte{})
				ids[2].SetValue(randomObjectID())
				r.SetFailSG(ids)
			}},
			{name: "invalid failed SG/wrong length", err: "invalid failed storage group #1: invalid length 31", corrupt: func(r *apiaudit.DataAuditResult) {
				ids := make([]refs.ObjectID, 3)
				ids[0].SetValue(randomObjectID())
				ids[1].SetValue(make([]byte, 31))
				ids[2].SetValue(randomObjectID())
				r.SetFailSG(ids)
			}},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				r := anyValidAuditResult()
				var msg apiaudit.DataAuditResult
				require.NoError(t, msg.Unmarshal(r.Marshal()))

				testCase.corrupt(&msg)

				require.EqualError(t, r.Unmarshal(msg.StableMarshal(nil)), testCase.err)
			})
		}
	})
}

func randomObjectID() []byte {
	o := oidtest.ID()
	return o[:]
}
