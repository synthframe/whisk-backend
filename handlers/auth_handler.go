package handlers

import (
	"net/http"
	"synthframe-api/services"

	"github.com/gin-gonic/gin"
)

func RegisterHandler(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=6"`
			Name     string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "올바른 이메일과 6자 이상의 비밀번호를 입력해주세요"})
			return
		}

		token, user, err := authSvc.Register(req.Email, req.Password, req.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user":  gin.H{"id": user.ID, "email": user.Email, "name": user.Name},
		})
	}
}

func LoginHandler(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "이메일과 비밀번호를 입력해주세요"})
			return
		}

		token, user, err := authSvc.Login(req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user":  gin.H{"id": user.ID, "email": user.Email, "name": user.Name},
		})
	}
}

func MeHandler(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		user, err := authSvc.GetUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "사용자를 찾을 수 없습니다"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": user.ID, "email": user.Email, "name": user.Name})
	}
}
