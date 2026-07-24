package api

import "github.com/gin-gonic/gin"

// OK 成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{
		"code": 0,
		"data": data,
	})
}

// Fail 失败响应
func Fail(c *gin.Context, code int, message string) {
	c.JSON(200, gin.H{
		"code":    code,
		"message": message,
	})
}

// Conflict returns a real HTTP conflict while preserving the API envelope.
func Conflict(c *gin.Context, message string, data interface{}) {
	c.JSON(409, gin.H{
		"code":    409,
		"message": message,
		"data":    data,
	})
}

// Warn 带警告但仍创建成功的响应（用于计划超负荷校验等场景）
func Warn(c *gin.Context, data interface{}, warnings []string) {
	c.JSON(200, gin.H{
		"code":     0,
		"data":     data,
		"warnings": warnings,
	})
}
