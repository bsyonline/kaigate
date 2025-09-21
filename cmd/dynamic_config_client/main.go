package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// 配置示例，用于更新config.yaml
const configExample = `
server:
  http_addr: ":8080"
  ws_addr: ":8081"
  admin_addr: ":8082"
  debug: true

log:
  level: "debug"
  format: "text"
  file: "logs/kaigate.log"
  stdout: true

websocket:
  heartbeat_interval: 30
  max_connections: 1000

router:
  enable_rate_limit: true
  default_rate_limit: 100
  circuit_break: true
  circuit_break_threshold: 50

proxy_routes:
  - path: "/deepsearch"
    target_url: "http://localhost:8301"
    enable: true
  - path: "/search"
    target_url: "http://localhost:8302"
    enable: true
  - path: "/api/external"
    target_url: "http://localhost:8303"
    enable: false
  - path: "/sandbox"
    target_url: "http://localhost:8302/sandbox"
    enable: true
`

func main() {
	// 打印程序使用说明
	fmt.Println("动态服务发现和注册演示客户端")
	fmt.Println("================================")
	fmt.Println("这个程序演示如何通过管理接口动态更新代理路由配置")
	fmt.Println("不需要重启服务就能接入新的服务端点")
	fmt.Println()

	// 读取命令行参数
	if len(os.Args) < 2 {
		fmt.Println("使用方法: go run main.go [command]")
	fmt.Println("commands:")
	fmt.Println("  update-config - 更新配置文件，添加新的/sandbox路由")
	fmt.Println("  reload-config - 调用管理接口重载整个配置")
	fmt.Println("  reload-routes - 调用管理接口仅重载代理路由")
	fmt.Println("  get-status - 获取当前服务状态和已注册的路由")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "update-config":
		updateConfigFile()
	case "reload-config":
		reloadConfig()
	case "reload-routes":
		reloadProxyRoutes()
	case "get-status":
		getStatus()
	default:
		fmt.Println("未知命令")
		os.Exit(1)
	}
}

// 更新配置文件，添加新的路由
func updateConfigFile() {
	// 配置文件路径
	configPath := "../../config.yaml"

	// 备份原始配置文件
	backupPath := configPath + ".bak." + time.Now().Format("20060102_150405")
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		return
	}

	err = ioutil.WriteFile(backupPath, content, 0644)
	if err != nil {
		fmt.Printf("备份配置文件失败: %v\n", err)
		return
	}

	// 写入新的配置文件
	err = ioutil.WriteFile(configPath, []byte(configExample), 0644)
	if err != nil {
		fmt.Printf("更新配置文件失败: %v\n", err)
		return
	}

	fmt.Println("配置文件已更新，添加了新的/sandbox路由")
	fmt.Printf("原始配置已备份到: %s\n", backupPath)
	fmt.Println("请运行 'go run main.go reload-routes' 来重载代理路由配置")
}

// 调用管理接口重载整个配置
func reloadConfig() {
	url := "http://localhost:8082/reload-config"

	// 发送POST请求
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Printf("发送请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	// 格式化JSON响应
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		fmt.Println("响应:", string(body))
	} else {
		fmt.Println("响应:", prettyJSON.String())
	}
}

// 调用管理接口仅重载代理路由
func reloadProxyRoutes() {
	url := "http://localhost:8082/reload-proxy-routes"

	// 发送POST请求
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Printf("发送请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	// 格式化JSON响应
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		fmt.Println("响应:", string(body))
	} else {
		fmt.Println("响应:", prettyJSON.String())
	}
}

// 获取当前服务状态和已注册的路由
func getStatus() {
	url := "http://localhost:8082/status"

	// 发送GET请求
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("发送请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	// 格式化JSON响应
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		fmt.Println("响应:", string(body))
	} else {
		fmt.Println("响应:", prettyJSON.String())
	}
}