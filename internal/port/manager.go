package port

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// PortManager 端口管理器
type PortManager struct {
	mu        sync.RWMutex
	minPort   int          // 最小端口号
	maxPort   int          // 最大端口号
	usedPorts map[int]bool // 已使用的端口
}

// NewPortManager 创建新的端口管理器
func NewPortManager(minPort, maxPort int) *PortManager {
	return &PortManager{
		minPort:   minPort,
		maxPort:   maxPort,
		usedPorts: make(map[int]bool),
	}
}

// IsPortAvailable 检查端口是否可用
func (pm *PortManager) IsPortAvailable(port int) bool {
	// 检查端口范围
	if port < pm.minPort || port > pm.maxPort {
		return false
	}

	// 检查是否已被标记为使用
	pm.mu.RLock()
	used := pm.usedPorts[port]
	pm.mu.RUnlock()
	if used {
		return false
	}

	// 通过系统调用检查端口是否真正可用
	return pm.checkPortBySocket(port)
}

// checkPortBySocket 通过socket检查端口是否可用
func (pm *PortManager) checkPortBySocket(port int) bool {
	// 尝试监听TCP端口
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()

	// 尝试监听UDP端口
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return false
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return false
	}
	udpConn.Close()

	return true
}

// GetRandomPort 获取一个随机可用端口
func (pm *PortManager) GetRandomPort() (int, error) {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// 最多尝试100次
	for i := 0; i < 100; i++ {
		port := rand.Intn(pm.maxPort-pm.minPort+1) + pm.minPort
		if pm.IsPortAvailable(port) {
			pm.markPortAsUsed(port)
			return port, nil
		}
	}

	return 0, fmt.Errorf("无法找到可用端口")
}

// GetRandomPorts 获取N个随机可用端口
func (pm *PortManager) GetRandomPorts(count int) ([]int, error) {
	if count <= 0 {
		return nil, fmt.Errorf("端口数量必须大于0")
	}

	ports := make([]int, 0, count)
	rand.Seed(time.Now().UnixNano())

	// 最多尝试count*100次
	for i := 0; i < count*100 && len(ports) < count; i++ {
		port := rand.Intn(pm.maxPort-pm.minPort+1) + pm.minPort
		if pm.IsPortAvailable(port) {
			// 检查是否已经在结果中
			alreadyExists := false
			for _, p := range ports {
				if p == port {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				pm.markPortAsUsed(port)
				ports = append(ports, port)
			}
		}
	}

	if len(ports) < count {
		return ports, fmt.Errorf("只找到%d个可用端口，需要%d个", len(ports), count)
	}

	return ports, nil
}

// markPortAsUsed 标记端口为已使用
func (pm *PortManager) markPortAsUsed(port int) {
	pm.mu.Lock()
	pm.usedPorts[port] = true
	pm.mu.Unlock()
}

// ReleasePort 释放端口
func (pm *PortManager) ReleasePort(port int) {
	pm.mu.Lock()
	delete(pm.usedPorts, port)
	pm.mu.Unlock()
}

// SetPortRange 设置端口范围
func (pm *PortManager) SetPortRange(minPort, maxPort int) error {
	if minPort <= 0 || maxPort <= 0 || minPort > maxPort {
		return fmt.Errorf("无效的端口范围: %d-%d", minPort, maxPort)
	}

	pm.mu.Lock()
	pm.minPort = minPort
	pm.maxPort = maxPort
	pm.mu.Unlock()

	return nil
}

// GetPortRange 获取端口范围
func (pm *PortManager) GetPortRange() (int, int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.minPort, pm.maxPort
}
