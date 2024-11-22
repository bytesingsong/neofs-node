package control

import (
	"context"
	"fmt"
	"os"

	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/shard"
	"github.com/nspcc-dev/neofs-node/pkg/services/control"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) DumpShard(_ context.Context, req *control.DumpShardRequest) (*control.DumpShardResponse, error) {
	err := s.isValidRequest(req)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	// check availability
	err = s.ready()
	if err != nil {
		return nil, err
	}

	shardID := shard.NewIDFromBytes(req.GetBody().GetShard_ID())

	f, err := os.OpenFile(req.GetBody().GetFilepath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return nil, fmt.Errorf("can't open destination file: %w", err)
	}
	defer f.Close()

	err = s.storage.DumpShard(shardID, f, req.GetBody().GetIgnoreErrors())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := new(control.DumpShardResponse)
	resp.SetBody(new(control.DumpShardResponse_Body))

	err = SignMessage(s.key, resp)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}
