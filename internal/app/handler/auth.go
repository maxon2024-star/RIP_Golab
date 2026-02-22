package handler

import (
	"RIP_Golab/internal/app/ds"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// Register - регистрация нового пользователя
func (h *Handler) Register(c *gin.Context) {
	var input struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}
	var existing ds.User
	if err := h.repo.GetDB().Where("login = ?", input.Login).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь уже существует"})
		return
	}
	user := ds.User{
		Login:       input.Login,
		Password:    input.Password,
		IsModerator: false,
	}
	if err := h.repo.GetDB().Create(&user).Error; err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}
	// ✅ Устанавливаем куки с userID
	c.SetCookie("user_id", strconv.Itoa(int(user.ID)), 3600*24*7, "/", "", false, true)
	c.SetCookie("user_login", user.Login, 3600*24*7, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"login":   user.Login,
	})
}

// Login - вход пользователя
func (h *Handler) Login(c *gin.Context) {
	var input struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}
	var user ds.User
	if err := h.repo.GetDB().Where("login = ? AND password = ?", input.Login, input.Password).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}
	// ✅ Устанавливаем куки с userID
	c.SetCookie("user_id", strconv.Itoa(int(user.ID)), 3600*24*7, "/", "", false, true)
	c.SetCookie("user_login", user.Login, 3600*24*7, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"user_id":      user.ID,
		"login":        user.Login,
		"is_moderator": user.IsModerator,
	})
}

// Logout - выход
func (h *Handler) Logout(c *gin.Context) {
	// ✅ Удаляем куки (устанавливаем maxAge = -1)
	c.SetCookie("user_id", "", -1, "/", "", false, true)
	c.SetCookie("user_login", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

// GetUser - получить текущего пользователя
func (h *Handler) GetUser(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}
	var user ds.User
	if err := h.repo.GetDB().First(&user, userID).Error; err != nil {
		h.errorHandler(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"login":        user.Login,
		"is_moderator": user.IsModerator,
	})
}
