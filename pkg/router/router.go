package router

import (
	"errors"
	"math/rand"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"kai/kaigate/pkg/log"
)

// Route 路由定义
type Route struct {
	ID          string            `json:"id"`
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	ServiceName string            `json:"service_name"`
	BackendURL  string            `json:"backend_url"`
	Weight      int               `json:"weight"`
	Headers     map[string]string `json:"headers"`
	Enabled     bool              `json:"enabled"`
}

// Router 路由管理器
type Router struct {
	routes         map[string][]*Route
	routesMutex    sync.RWMutex
	rateLimiters   map[string]*RateLimiter
	circuitBreaker *CircuitBreaker
}

// NewRouter 创建路由管理器
func NewRouter() *Router {
	router := &Router{
		routes:         make(map[string][]*Route),
		rateLimiters:   make(map[string]*RateLimiter),
		circuitBreaker: NewCircuitBreaker(),
	}

	// 初始化默认路由
	router.initDefaultRoutes()

	return router
}

// initDefaultRoutes 初始化默认路由
func (r *Router) initDefaultRoutes() {
	// 添加一些默认的测试路由
	r.AddRoute(&Route{
		ID:          "test-echo",
		Path:        "/api/v1/echo",
		Method:      "GET",
		ServiceName: "test",
		BackendURL:  "http://localhost:8080",
		Weight:      100,
		Enabled:     true,
	})
}

// AddRoute 添加路由
func (r *Router) AddRoute(route *Route) error {
	if route == nil {
		return errors.New("route cannot be nil")
	}

	if route.Path == "" {
		return errors.New("route path cannot be empty")
	}

	if route.Method == "" {
		return errors.New("route method cannot be empty")
	}

	if route.ServiceName == "" {
		return errors.New("route service name cannot be empty")
	}

	// 生成路由键
	routeKey := route.Method + "-" + route.Path

	// 添加路由
	r.routesMutex.Lock()
	defer r.routesMutex.Unlock()

	// 检查路由是否已存在
	for _, r := range r.routes[routeKey] {
		if r.ID == route.ID {
			return errors.New("route with same ID already exists")
		}
	}

	// 添加路由到列表
	r.routes[routeKey] = append(r.routes[routeKey], route)

	// 记录日志
	log.GlobalLogger.Info("Route added",
		zap.String("id", route.ID),
		zap.String("path", route.Path),
		zap.String("method", route.Method),
		zap.String("service_name", route.ServiceName),
	)

	return nil
}

// RemoveRoute 移除路由
func (r *Router) RemoveRoute(routeID string) error {
	r.routesMutex.Lock()
	defer r.routesMutex.Unlock()

	// 查找并移除路由
	for key, routes := range r.routes {
		newRoutes := make([]*Route, 0)
		found := false
		for _, route := range routes {
			if route.ID != routeID {
				newRoutes = append(newRoutes, route)
			} else {
				found = true
				log.GlobalLogger.Info("Route removed",
					zap.String("id", route.ID),
					zap.String("path", route.Path),
					zap.String("method", route.Method),
				)
			}
		}
		r.routes[key] = newRoutes
		// 如果路由列表为空，则删除该键
		if len(newRoutes) == 0 {
			delete(r.routes, key)
		}
		if found {
			return nil
		}
	}

	return errors.New("route not found")
}

// GetRoute 获取路由
func (r *Router) GetRoute(method, path string) ([]*Route, bool) {
	// 生成路由键
	routeKey := method + "-" + path

	// 查找路由
	r.routesMutex.RLock()
	defer r.routesMutex.RUnlock()
	routes, ok := r.routes[routeKey]
	return routes, ok
}

// GetAllRoutes 获取所有路由
func (r *Router) GetAllRoutes() []*Route {
	r.routesMutex.RLock()
	defer r.routesMutex.RUnlock()

	allRoutes := make([]*Route, 0)
	for _, routes := range r.routes {
		allRoutes = append(allRoutes, routes...)
	}

	return allRoutes
}

// UpdateRoute 更新路由
func (r *Router) UpdateRoute(route *Route) error {
	if route == nil {
		return errors.New("route cannot be nil")
	}

	// 先移除旧路由
	if err := r.RemoveRoute(route.ID); err != nil {
		return err
	}

	// 添加新路由
	return r.AddRoute(route)
}

// MatchRoute 匹配路由
func (r *Router) MatchRoute(req *http.Request) (*Route, bool) {
	// 获取路由
	method := req.Method
	path := req.URL.Path
	routes, ok := r.GetRoute(method, path)
	if !ok || len(routes) == 0 {
		return nil, false
	}

	// 过滤启用的路由
	enabledRoutes := make([]*Route, 0)
	for _, route := range routes {
		if route.Enabled {
			enabledRoutes = append(enabledRoutes, route)
		}
	}

	if len(enabledRoutes) == 0 {
		return nil, false
	}

	// 使用负载均衡算法选择路由
	selectedRoute := r.selectRoute(enabledRoutes)

	return selectedRoute, true
}

// selectRoute 选择路由（负载均衡）
func (r *Router) selectRoute(routes []*Route) *Route {
	// 简单的加权轮询算法
	totalWeight := 0
	for _, route := range routes {
		totalWeight += route.Weight
	}

	if totalWeight == 0 {
		// 如果总权重为0，随机选择一个
		return routes[rand.Intn(len(routes))]
	}

	// 随机生成一个0到总权重之间的数
	random := rand.Intn(totalWeight)

	// 根据权重选择路由
	current := 0
	for _, route := range routes {
		current += route.Weight
		if random < current {
			return route
		}
	}

	// 兜底返回第一个
	return routes[0]
}
