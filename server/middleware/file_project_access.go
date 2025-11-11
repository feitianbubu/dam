package middleware

import (
	"fmt"
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// FileProjectAccessHandler 文件项目权限控制中间件
// 权限模型：read（读）和 write（写/删除）
// 公开项目：GET 请求支持匿名访问
// 私有项目：所有操作需要 JWT + Casbin 权限验证
func FileProjectAccessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从请求中提取 projectId
		projectID := extractProjectID(c)
		if projectID == "" {
			// 如果没有 projectId，按原有逻辑处理（向后兼容）
			c.Next()
			return
		}

		// 2. 查询项目信息
		var project example.Project
		if err := global.GVA_DB.First(&project, "id = ?", projectID).Error; err != nil {
			response.FailWithMessage("项目不存在", c)
			c.Abort()
			return
		}

		// 3. 公开项目 + GET 请求 → 支持匿名访问
		if project.IsPublic && c.Request.Method == "GET" {
			c.Set("project", project) // 存储项目信息供后续使用
			c.Next()
			return
		}

		// 4. 私有项目或写操作 → 需要 JWT 认证
		claims, err := utils.GetClaims(c)
		if err != nil {
			response.FailWithMessage("未登录或token无效", c)
			c.Abort()
			return
		}

		// 5. Casbin 权限验证
		sub := strconv.Itoa(int(claims.AuthorityId))
		obj := fmt.Sprintf("/project/%s", projectID)
		act := mapMethodToPermission(c.Request.Method) // GET→read, 其他→write

		e := utils.GetCasbin()
		success, _ := e.Enforce(sub, obj, act)

		if !success {
			response.FailWithMessage(fmt.Sprintf("无权限访问此项目(%s)", act), c)
			c.Abort()
			return
		}

		c.Set("project", project)
		c.Next()
	}
}

// extractProjectID 从多个来源提取 projectId
// 优先级：URL参数 > Query参数 > POST表单 > JSON body
func extractProjectID(c *gin.Context) string {
	// 1. URL 参数（如 /project/:projectId/...）
	if id := c.Param("projectId"); id != "" {
		return id
	}

	// 2. Query 参数（如 ?projectId=xxx）
	if id := c.Query("projectId"); id != "" {
		return id
	}

	// 3. POST 表单（如 multipart/form-data）
	if id := c.PostForm("projectId"); id != "" {
		return id
	}

	// 4. JSON body（如果是 application/json 请求）
	var body map[string]interface{}
	if err := c.ShouldBindBodyWith(&body, binding.JSON); err == nil {
		if id, ok := body["projectId"].(string); ok {
			return id
		}
	}

	return ""
}

// mapMethodToPermission 映射 HTTP 方法到权限类型
// read: GET
// write: POST, PUT, PATCH, DELETE
func mapMethodToPermission(method string) string {
	switch method {
	case "GET":
		return "read"
	case "POST", "PUT", "PATCH", "DELETE":
		return "write"
	default:
		return "read"
	}
}
