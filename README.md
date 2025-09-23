# Themis - GRPC端口管理服务

Themis是一个基于Go语言开发的GRPC端口管理服务，提供端口分配、管理和持久化功能。

## 功能特性

- **端口检测**: 通过系统调用检测端口可用性，支持TCP和UDP端口检测
- **端口范围配置**: 支持自定义端口范围，默认10000-20000
- **应用端口映射**: 维护应用名称到端口的全局映射字典
- **数据持久化**: 支持定时同步到JSON文件，启动时自动加载
- **GRPC接口**: 提供完整的GRPC服务接口
- **灵活监听**: 支持Unix Socket和TCP两种监听方式

## 项目结构

```
Themis/
├── cmd/server/          # 主程序入口
├── configs/             # 配置文件
├── internal/            # 内部模块
│   ├── config/         # 配置管理
│   ├── port/           # 端口管理
│   ├── server/         # GRPC服务实现
│   └── storage/        # 数据存储
├── proto/              # Protocol Buffers定义
├── data/               # 数据文件目录（运行时创建）
├── go.mod              # Go模块文件
└── requirements.md     # 需求文档
```

## 快速开始

### 1. 编译项目

```bash
go build -o themis ./cmd/server
```

### 2. 运行服务

```bash
# 使用默认配置
./themis

# 指定配置文件
./themis -config ./configs/config.json
```

### 3. 配置说明

配置文件采用JSON格式，主要包含以下配置项：

```json
{
  "server": {
    "listen_type": "unix",           // 监听类型: "unix" 或 "tcp"
    "unix_socket": "/var/run/Themis.sock", // Unix socket路径
    "host": "localhost",             // TCP监听主机
    "port": 9090                     // TCP监听端口
  },
  "port": {
    "min_port": 10000,               // 最小端口号
    "max_port": 20000                // 最大端口号
  },
  "storage": {
    "data_file": "./data/ports.json", // 数据文件路径
    "sync_interval": 60              // 同步间隔（秒）
  }
}
```

## GRPC接口

### 1. 获取随机端口

```protobuf
rpc GetRandomPort(GetRandomPortRequest) returns (GetRandomPortResponse);
```

### 2. 获取N个随机端口

```protobuf
rpc GetRandomPorts(GetRandomPortsRequest) returns (GetRandomPortsResponse);
```

### 3. 获取应用端口

```protobuf
rpc GetAppPort(GetAppPortRequest) returns (GetAppPortResponse);
```

### 4. 设置应用端口

```protobuf
rpc SetAppPort(SetAppPortRequest) returns (SetAppPortResponse);
```

### 5. 快速设置应用端口

```protobuf
rpc QuickSetAppPort(QuickSetAppPortRequest) returns (QuickSetAppPortResponse);
```

### 6. 删除应用端口

```protobuf
rpc DeleteAppPort(DeleteAppPortRequest) returns (DeleteAppPortResponse);
```

## 使用示例

### Go客户端示例

```go
package main

import (
    "context"
    "log"
    
    "google.golang.org/grpc"
    pb "github.com/JJApplication/Themis/proto"
)

func main() {
    // 连接到服务
    conn, err := grpc.Dial("unix:///var/run/Themis.sock", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    client := pb.NewPortServiceClient(conn)
    
    // 获取随机端口
    resp, err := client.GetRandomPort(context.Background(), &pb.GetRandomPortRequest{})
    if err != nil {
        log.Fatal(err)
    }
    
    if resp.Error != "" {
        log.Printf("错误: %s", resp.Error)
    } else {
        log.Printf("获取到端口: %d", resp.Port)
    }
}
```

## 数据持久化

服务会定期将应用端口映射数据同步到JSON文件中，文件格式如下：

```json
{
  "app_ports": {
    "app1": 10001,
    "app2": 10002
  },
  "timestamp": 1640995200,
  "version": "1.0"
}
```

## 安全考虑

- Unix Socket默认权限为644，建议根据实际需求调整
- 数据文件建议设置适当的文件权限
- 生产环境建议使用TLS加密（TCP模式）

## 性能特性

- 并发安全的端口管理
- 高效的端口可用性检测
- 异步数据持久化
- 优雅的服务关闭

## 许可证

本项目采用MIT许可证，详见LICENSE文件。