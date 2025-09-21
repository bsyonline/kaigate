package main

import (
	"fmt"
	"encoding/json"
	"kai/kaigate/pkg/config"
)

func main() {
	// 初始化配置，使用当前目录下的config.yaml
	if err := config.InitConfig("config.yaml"); err != nil {
		fmt.Printf("Failed to initialize configuration: %v\n", err)
		return
	}

	// 输出整个配置
	configJSON, _ := json.MarshalIndent(config.GlobalConfig, "", "  ")
	fmt.Println("Loaded configuration:")
	fmt.Println(string(configJSON))

	// 专门输出ProxyRoutes配置
	fmt.Println("\nProxy Routes configuration:")
	for i, route := range config.GlobalConfig.ProxyRoutes {
		fmt.Printf("Route %d: Path=%s, TargetURL=%s, Enable=%v\n", 
			i, route.Path, route.TargetURL, route.Enable)
	}
}