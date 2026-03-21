package handler

import (
	"RIP_Golab/internal/app/role"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt"
)

const jwtSecret = "super_secret_physics_key_123" // Секрет для подписи JWT
const jwtPrefix = "Bearer "

// Структура полезной нагрузки JWT
type JWTClaims struct {
	jwt.StandardClaims
	PhysicistID uint      `json:"physicist_id"`
	Role        role.Role `json:"role"`
}

// Middleware проверки JWT и Blacklist'а в Redis
func (h *Handler) WithAuthCheck(allowedRoles ...role.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, jwtPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Необходим Bearer токен авторизации"})
			return
		}

		tokenStr := authHeader[len(jwtPrefix):]

		// 1. Проверяем токен в Blacklist (Redis)
		err := h.repo.GetRedis().Get(c.Request.Context(), "blacklist:"+tokenStr).Err()
		if err == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Токен недействителен (Вы вышли из системы)"})
			return
		} else if err != redis.Nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки токена в Redis"})
			return
		}

		// 2. Парсим и валидируем JWT
		token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Неверный или просроченный токен"})
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Ошибка чтения данных токена"})
			return
		}

		// 3. Проверка ролевого доступа
		hasAccess := false
		for _, r := range allowedRoles {
			if claims.Role == r || claims.Role == role.Professor {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Доступ запрещен для вашей роли"})
			return
		}

		// Сохраняем данные пользователя в контекст Gin
		c.Set("physicist_id", claims.PhysicistID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// Получение ID Физика из контекста (заменяет старый CurrentUser)
func (h *Handler) getPhysicistID(c *gin.Context) uint {
	id, exists := c.Get("physicist_id")
	if exists {
		return id.(uint)
	}
	return 0 // Гость
}
