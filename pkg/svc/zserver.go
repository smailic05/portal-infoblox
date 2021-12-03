package svc

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/smailic05/portal-infoblox/pkg/pb"
	rpb "github.com/smailic05/portal-infoblox/pkg/resp_pb"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// version is the current version of the service
	version = "0.0.1"
)

// Default implementation of the MyApp server interface
type server struct {
	pb.UnimplementedMyAppServer
	Description string
	Timestamp   time.Time
	Requests    int64
	mtxRequests sync.RWMutex
	mtxDescr    sync.RWMutex
}

func GRPCConnect() (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	addr := viper.GetString("server.address.full")
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// GetVersion returns the current version of the service
func (s *server) GetVersion(context.Context, *empty.Empty) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{Version: version}, nil
	// TODO use hash commit as version
}

func (s *server) UpdateDescription(ctx context.Context, req *pb.UpdateDescriptionRequest) (*pb.UpdateDescriptionResponse, error) {
	if req.GetDescription() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Description can't be empty")
	}
	if req.GetService() != 1 {
		conn, err := GRPCConnect()
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := rpb.NewMyResponderClient(conn)
		resp, err := client.UpdateDescription(ctx, &rpb.UpdateDescriptionRequest{Description: req.Description, Service: req.Service})
		if err != nil {
			return nil, err
		}
		return &pb.UpdateDescriptionResponse{Description: resp.Description}, nil
	}
	s.IncRequests()
	s.UpdateDescriptionFromServer(req.Description)
	return &pb.UpdateDescriptionResponse{Description: s.GetDescriptionFromServer()}, nil
}

func (s *server) GetDescription(ctx context.Context, req *pb.GetDescriptionRequest) (*pb.GetDescriptionResponse, error) {
	if req.GetService() != 1 {
		conn, err := GRPCConnect()
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := rpb.NewMyResponderClient(conn)
		resp, err := client.GetDescription(ctx, &rpb.GetDescriptionRequest{Service: req.Service})
		if err != nil {
			return nil, err
		}
		return &pb.GetDescriptionResponse{Description: resp.Description}, nil
	}
	s.IncRequests()
	return &pb.GetDescriptionResponse{Description: s.GetDescriptionFromServer()}, nil
}

// Returns uptime in seconds
func (s *server) GetUptime(ctx context.Context, req *pb.GetUptimeRequest) (*pb.GetUptimeResponse, error) {
	if req.GetService() != 1 {
		conn, err := GRPCConnect()
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := rpb.NewMyResponderClient(conn)
		resp, err := client.GetUptime(ctx, &rpb.GetUptimeRequest{Service: req.Service})
		if err != nil {
			return nil, err
		}
		return &pb.GetUptimeResponse{Uptime: resp.Uptime}, nil
	}
	s.IncRequests()
	uptime := time.Now().Unix() - s.Timestamp.Unix()
	return &pb.GetUptimeResponse{Uptime: uptime}, nil
}

func (s *server) GetRequests(ctx context.Context, req *pb.GetRequestsRequest) (*pb.GetRequestsResponse, error) {
	if req.GetService() != 1 {
		conn, err := GRPCConnect()
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := rpb.NewMyResponderClient(conn)
		resp, err := client.GetRequests(ctx, &rpb.GetRequestsRequest{Service: req.Service})
		if err != nil {
			return nil, err
		}
		return &pb.GetRequestsResponse{Requests: resp.Requests}, nil
	}
	s.IncRequests()
	return &pb.GetRequestsResponse{Requests: int64(s.GetRequestsFromServer())}, nil
}

func (s *server) GetMode(ctx context.Context, req *pb.GetModeRequest) (*pb.GetModeResponse, error) {
	conn, err := GRPCConnect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := rpb.NewMyResponderClient(conn)
	resp, err := client.GetMode(ctx, &rpb.GetModeRequest{})
	if err != nil {
		return nil, err
	}
	s.IncRequests()
	return &pb.GetModeResponse{Mode: resp.Mode}, nil
}

func (s *server) SetMode(ctx context.Context, req *pb.SetModeRequest) (*pb.SetModeResponse, error) {
	conn, err := GRPCConnect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := rpb.NewMyResponderClient(conn)
	resp, err := client.SetMode(ctx, &rpb.SetModeRequest{Mode: req.Mode})
	if err != nil {
		return nil, err
	}
	s.IncRequests()
	return &pb.SetModeResponse{Mode: resp.Mode}, nil
}

func (s *server) Restart(ctx context.Context, req *pb.RestartRequest) (*pb.RestartResponse, error) {
	if req.Service != 1 {
		conn, err := GRPCConnect()
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := rpb.NewMyResponderClient(conn)
		_, err = client.Restart(ctx, &rpb.RestartRequest{Service: req.Service})
		if err != nil {
			return nil, err
		}
		return &pb.RestartResponse{}, nil
	}
	s.Description = viper.GetString("app.id")
	s.Timestamp = time.Now()
	s.Requests = 0
	return &pb.RestartResponse{}, nil
}

// NewBasicServer returns an instance of the default server interface
func NewBasicServer() (pb.MyAppServer, error) {
	return &server{Description: viper.GetString("app.id"), Timestamp: time.Now()}, nil
}

func (s *server) GetRequestsFromServer() int {
	s.mtxRequests.RLock()
	defer s.mtxRequests.RUnlock()
	return int(s.Requests)
}

func (s *server) IncRequests() {
	s.mtxRequests.Lock()
	s.Requests++
	s.mtxRequests.Unlock()
}

func (s *server) GetDescriptionFromServer() string {
	s.mtxDescr.RLock()
	defer s.mtxDescr.RUnlock()
	return s.Description
}

func (s *server) UpdateDescriptionFromServer(description string) {
	s.mtxDescr.Lock()
	s.Description = description
	s.mtxDescr.Unlock()
}
