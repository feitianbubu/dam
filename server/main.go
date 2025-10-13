package main

import (
	"github.com/flipped-aurora/gin-vue-admin/server/core"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/initialize"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

// 这部分 @Tag 设置用于排序, 需要排序的接口请按照下面的格式添加
// swag init 对 @Tag 只会从入口文件解析, 默认 main.go
// 也可通过 --generalInfo flag 指定其他文件
// @Tag.Name        Base
// @Tag.Name        SysUser
// @Tag.Description 用户

// @title        云一Dam API接口文档
// @version      {{.Version}}
// @description  ###云一AI图像视频数据管理系统DAM
// @description
// @description  **调用流程:**
// @description  1. 获取token
// @description  2. 在请求头携带x-token
// @description  3. 调用业务接口
// @description
// @description  **获取token方式(三选一):**
// @description  1. 账号密码登录获取(简单联调使用)
// @description  2. 通过Oidc标准登录获取(推荐) → 调用`获取Oidc授权URL`Api → 跳转到授权URL登录 → 登录成功后获取授权码code → 通过code调用`Oidc回调处理`Api获取token
// @description  3. 通过Clinx登录(接过Clinx) → 获取Clinx访问令牌access_token → 通过access_token调用`Oidc回调处理`Api获取token
// @description
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        x-token
// @BasePath                    /
func main() {
	// 初始化系统
	initializeSystem()
	// 运行服务器
	core.RunServer()
}

// initializeSystem 初始化系统所有组件
// 提取为单独函数以便于系统重载时调用
func initializeSystem() {
	global.GVA_VP = core.Viper() // 初始化Viper
	initialize.OtherInit()
	global.GVA_LOG = core.Zap() // 初始化zap日志库
	zap.ReplaceGlobals(global.GVA_LOG)
	global.GVA_DB = initialize.Gorm() // gorm连接数据库
	initialize.Timer()
	initialize.DBList()
	initialize.SetupHandlers() // 注册全局函数
	if global.GVA_DB != nil {
		initialize.RegisterTables() // 初始化表
	}
}
