package main

import (
	"flag"
	"log"
	"os"

	"github.com/JJApplication/Themis/internal/config"
	"github.com/JJApplication/Themis/internal/server"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "config.json", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	// 创建并启动服务
	portServer := server.NewPortServer(cfg)
	if err := portServer.Start(); err != nil {
		log.Fatalf("启动服务失败: %v", err)
		os.Exit(1)
	}
}
