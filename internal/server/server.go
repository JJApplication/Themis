package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JJApplication/Themis/internal/config"
	"github.com/JJApplication/Themis/internal/port"
	"github.com/JJApplication/Themis/internal/storage"
	pb "github.com/JJApplication/Themis/proto"
	"google.golang.org/grpc"
)

// PortServer GRPC端口服务实现
type PortServer struct {
	pb.UnimplementedPortServiceServer
	portManager *port.PortManager       // 端口管理器
	storage     *storage.AppPortStorage // 存储管理器
	config      *config.Config          // 配置
}

// NewPortServer 创建新的端口服务
func NewPortServer(cfg *config.Config) *PortServer {
	// 创建端口管理器
	portManager := port.NewPortManager(cfg.Port.MinPort, cfg.Port.MaxPort)

	// 创建存储管理器
	syncInterval := time.Duration(cfg.Storage.SyncInterval) * time.Second
	storageManager := storage.NewAppPortStorage(cfg.Storage.DataFile, syncInterval)

	return &PortServer{
		portManager: portManager,
		storage:     storageManager,
		config:      cfg,
	}
}

// Start 启动服务
func (s *PortServer) Start() error {
	// 加载存储数据
	if err := s.storage.LoadFromFile(); err != nil {
		return fmt.Errorf("加载存储数据失败: %v", err)
	}

	// 启动自动同步
	s.storage.StartAutoSync()

	// 创建gRPC服务器
	grpcServer := grpc.NewServer()
	pb.RegisterPortServiceServer(grpcServer, s)

	// 创建监听器
	listener, err := s.createListener()
	if err != nil {
		return fmt.Errorf("创建监听器失败: %v", err)
	}

	log.Printf("端口服务启动，监听地址: %s", s.config.GetListenAddress())

	// 启动服务器
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC服务器错误: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭服务...")

	// 优雅关闭
	grpcServer.GracefulStop()
	s.storage.StopAutoSync()

	log.Println("服务已关闭")
	return nil
}

// createListener 创建监听器
func (s *PortServer) createListener() (net.Listener, error) {
	if s.config.Server.ListenType == "unix" {
		if s.config.Server.UnixSocket == "" {
			return nil, fmt.Errorf("unix socket路径不能为空")
		}
		// 删除已存在的socket文件
		if err := os.RemoveAll(s.config.Server.UnixSocket); err != nil {
			return nil, fmt.Errorf("删除socket文件失败: %v", err)
		}
		return net.Listen("unix", s.config.Server.UnixSocket)
	} else {
		addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
		return net.Listen("tcp", addr)
	}
}

// GetRandomPort 获取一个随机端口
func (s *PortServer) GetRandomPort(ctx context.Context, req *pb.GetRandomPortRequest) (*pb.GetRandomPortResponse, error) {
	port, err := s.portManager.GetRandomPort()
	if err != nil {
		return &pb.GetRandomPortResponse{
			Port:  0,
			Error: err.Error(),
		}, nil
	}

	return &pb.GetRandomPortResponse{
		Port:  int32(port),
		Error: "",
	}, nil
}

// GetRandomPorts 获取N个随机端口
func (s *PortServer) GetRandomPorts(ctx context.Context, req *pb.GetRandomPortsRequest) (*pb.GetRandomPortsResponse, error) {
	if req.Count <= 0 {
		return &pb.GetRandomPortsResponse{
			Ports: nil,
			Error: "端口数量必须大于0",
		}, nil
	}

	ports, err := s.portManager.GetRandomPorts(int(req.Count))
	if err != nil {
		return &pb.GetRandomPortsResponse{
			Ports: nil,
			Error: err.Error(),
		}, nil
	}

	// 转换为int32数组
	int32Ports := make([]int32, len(ports))
	for i, port := range ports {
		int32Ports[i] = int32(port)
	}

	return &pb.GetRandomPortsResponse{
		Ports: int32Ports,
		Error: "",
	}, nil
}

// GetAppPort 获取某个APP的端口
func (s *PortServer) GetAppPort(ctx context.Context, req *pb.GetAppPortRequest) (*pb.GetAppPortResponse, error) {
	port, err := s.storage.GetAppPort(req.AppName)
	if err != nil {
		return &pb.GetAppPortResponse{
			Port:  0,
			Error: err.Error(),
		}, nil
	}

	return &pb.GetAppPortResponse{
		Port:  int32(port),
		Error: "",
	}, nil
}

// SetAppPort 设置某个APP的端口
func (s *PortServer) SetAppPort(ctx context.Context, req *pb.SetAppPortRequest) (*pb.SetAppPortResponse, error) {
	err := s.storage.SetAppPort(req.AppName, int(req.Port))
	if err != nil {
		return &pb.SetAppPortResponse{
			Error: err.Error(),
		}, nil
	}

	return &pb.SetAppPortResponse{
		Error: "",
	}, nil
}

// QuickSetAppPort 快速设置APP端口（存在则返回，不存在则生成随机端口）
func (s *PortServer) QuickSetAppPort(ctx context.Context, req *pb.QuickSetAppPortRequest) (*pb.QuickSetAppPortResponse, error) {
	// 检查APP是否已存在
	if s.storage.HasApp(req.AppName) {
		port, err := s.storage.GetAppPort(req.AppName)
		if err != nil {
			return &pb.QuickSetAppPortResponse{
				Port:  0,
				Error: err.Error(),
			}, nil
		}
		return &pb.QuickSetAppPortResponse{
			Port:  int32(port),
			Error: "",
		}, nil
	}

	// 生成随机端口
	port, err := s.portManager.GetRandomPort()
	if err != nil {
		return &pb.QuickSetAppPortResponse{
			Port:  0,
			Error: err.Error(),
		}, nil
	}

	// 设置APP端口
	if err := s.storage.SetAppPort(req.AppName, port); err != nil {
		return &pb.QuickSetAppPortResponse{
			Port:  0,
			Error: err.Error(),
		}, nil
	}

	return &pb.QuickSetAppPortResponse{
		Port:  int32(port),
		Error: "",
	}, nil
}

// DeleteAppPort 删除某个服务的端口信息
func (s *PortServer) DeleteAppPort(ctx context.Context, req *pb.DeleteAppPortRequest) (*pb.DeleteAppPortResponse, error) {
	if req.AppName == "" {
		return &pb.DeleteAppPortResponse{
			Error: "APP名称不能为空",
		}, nil
	}

		// 获取端口以便释放
	if port, err := s.storage.GetAppPort(req.AppName); err == nil {
		s.portManager.ReleasePort(port)
	}

	// 删除APP端口信息
	err := s.storage.DeleteAppPort(req.AppName)
	if err != nil {
		return &pb.DeleteAppPortResponse{
			Error: err.Error(),
		}, nil
	}

	return &pb.DeleteAppPortResponse{
		Error: "",
	}, nil
}

// IsPortAvailable 检查端口是否可用
func (s *PortServer) IsPortAvailable(ctx context.Context, req *pb.IsPortAvailableRequest) (*pb.IsPortAvailableResponse, error) {
	if req.Port <= 0 || req.Port > 65535 {
		return &pb.IsPortAvailableResponse{
			Available: false,
			Error:     "端口号必须在1-65535范围内",
		}, nil
	}

	// 调用端口管理器的IsPortAvailable方法检查端口可用性
	available := s.portManager.IsPortAvailable(int(req.Port))

	return &pb.IsPortAvailableResponse{
		Available: available,
		Error:     "",
	}, nil
}
