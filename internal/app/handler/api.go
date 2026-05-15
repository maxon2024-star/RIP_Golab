package handler

import (
	"RIP_Golab/internal/app/ds"
	"RIP_Golab/internal/app/role"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

// === ДОМЕН УСЛУГИ (ИЗЛУЧЕНИЯ) ===

// @Summary Получение всех услуг
func (h *Handler) GetRadiationsAPI(c *gin.Context) {
	search := c.Query("search")
	var radiations []ds.RadiationRange

	db := h.repo.GetDB().Where("is_delete = ?", false)
	if search != "" {
		db = db.Where("name ILIKE ?", "%"+search+"%")
	}
	db.Find(&radiations)

	c.JSON(http.StatusOK, radiations)
}

// @Summary Получение услуги по ID
func (h *Handler) GetRadiationAPI(c *gin.Context) {
	id := c.Param("id")
	var radiation ds.RadiationRange
	if err := h.repo.GetDB().First(&radiation, "id = ? AND is_delete = false", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Излучение не найдено"})
		return
	}
	c.JSON(http.StatusOK, radiation)
}

// @Summary Добавление новой услуги
func (h *Handler) AddRadiationAPI(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка формы"})
		return
	}

	radiation := ds.RadiationRange{
		Name:             c.PostForm("name"),
		ShortDescription: c.PostForm("short_description"),
		Description:      c.PostForm("description"),
	}

	uploadToMinio := func(field string) string {
		file, header, err := c.Request.FormFile(field)
		if err == nil {
			defer file.Close()
			ext := ".jpg"
			contentType := "image/jpeg"

			if strings.Contains(header.Filename, ".mp4") {
				ext = ".mp4"
				contentType = "video/mp4"
			} else if strings.Contains(header.Filename, ".png") {
				ext = ".png"
				contentType = "image/png"
			}

			filename := fmt.Sprintf("%s_%d%s", field, time.Now().Unix(), ext)
			bucket := "physicsservice"

			_, err = h.repo.GetMinio().PutObject(context.Background(), bucket, filename, file, header.Size, minio.PutObjectOptions{ContentType: contentType})
			if err != nil {
				logrus.Errorf("Ошибка загрузки в MinIO: %v", err)
				return ""
			}
			return "http://localhost:9000/" + bucket + "/" + filename
		}
		return ""
	}

	radiation.ImageURL = uploadToMinio("image")
	radiation.VideoURL = uploadToMinio("video")

	h.repo.GetDB().Create(&radiation)
	c.JSON(http.StatusCreated, radiation)
}

// === ДОМЕН М-М (ЭЛЕМЕНТЫ РАСЧЕТА) ===

type M2MInput struct {
	RadiationID  uint    `json:"radiation_id" binding:"required"`
	Area         float64 `json:"area"`
	Efficiency   float64 `json:"efficiency"`
	Frequency    float64 `json:"frequency"`
	WorkFunction float64 `json:"work_function"`
	IsPriority   bool    `json:"is_priority"` // <--- ИМЕННО ЗДЕСЬ ДОЛЖНА БЫТЬ ГАЛОЧКА!
}

// @Summary Добавление излучения в корзину (черновик)
func (h *Handler) AddCalculationItemAPI(c *gin.Context) {
	var input M2MInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Frequency <= 0 || input.WorkFunction <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Необходимо указать частоту (>0) и работу выхода (>0)"})
		return
	}

	physicistID := h.getPhysicistID(c)
	var draft ds.RadiationCalculation

	err := h.repo.GetDB().Where("physicist_id = ? AND status = 'draft'", physicistID).First(&draft).Error
	if err != nil {
		draft = ds.RadiationCalculation{PhysicistID: physicistID, Status: "draft"}
		h.repo.GetDB().Create(&draft)
	}

	item := ds.CalculationItem{
		CalculationID: draft.ID,
		RadiationID:   input.RadiationID,
		Area:          input.Area,
		Efficiency:    input.Efficiency,
		Frequency:     input.Frequency,
		WorkFunction:  input.WorkFunction,
		IsPriority:    input.IsPriority, // Сохраняем галочку
	}
	h.repo.GetDB().Create(&item)
	c.JSON(http.StatusCreated, gin.H{"message": "Излучение добавлено в расчет"})
}

// @Summary Обновление элемента в корзине
func (h *Handler) UpdateCalculationItemAPI(c *gin.Context) {
	var input M2MInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var draft ds.RadiationCalculation
	if err := h.repo.GetDB().Where("physicist_id = ? AND status = 'draft'", h.getPhysicistID(c)).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Черновик не найден"})
		return
	}

	var item ds.CalculationItem
	if err := h.repo.GetDB().Where("calculation_id = ? AND radiation_id = ?", draft.ID, input.RadiationID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Услуга не найдена в заявке"})
		return
	}

	// ЖЕСТКО обновляем все переданные поля напрямую в структуре
	item.Area = input.Area
	item.Efficiency = input.Efficiency
	item.Frequency = input.Frequency
	item.WorkFunction = input.WorkFunction
	item.IsPriority = input.IsPriority

	// Сохраняем всю строку целиком (GORM не проигнорирует bool)
	if err := h.repo.GetDB().Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения в БД"})
		return
	}

	c.Status(http.StatusOK)
}

// @Summary Удаление элемента из корзины
func (h *Handler) DeleteCalculationItemAPI(c *gin.Context) {
	radiationID := c.Query("radiation_id")

	var draft ds.RadiationCalculation
	if err := h.repo.GetDB().Where("physicist_id = ? AND status = 'draft'", h.getPhysicistID(c)).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Черновик не найден"})
		return
	}

	h.repo.GetDB().Where("calculation_id = ? AND radiation_id = ?", draft.ID, radiationID).Delete(&ds.CalculationItem{})
	c.Status(http.StatusNoContent)
}

// === ДОМЕН ЗАЯВКИ (РАСЧЕТЫ) ===

// @Summary Получение иконки корзины (сводка черновика)
func (h *Handler) GetDraftSummaryAPI(c *gin.Context) {
	physicistID := h.getPhysicistID(c)

	if physicistID == 0 {
		c.JSON(http.StatusOK, gin.H{"draft_id": nil, "count": 0})
		return
	}

	var draft ds.RadiationCalculation
	if err := h.repo.GetDB().Preload("Items").Where("physicist_id = ? AND status = 'draft'", physicistID).First(&draft).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"draft_id": nil, "count": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft_id": draft.ID, "count": len(draft.Items)})
}

// @Summary Список всех сформированных заявок (с фильтрами)
func (h *Handler) GetCalculationsAPI(c *gin.Context) {
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	status := c.Query("status")

	var calculations []ds.RadiationCalculation
	db := h.repo.GetDB().Preload("Physicist").Preload("Items").Where("status NOT IN ('draft', 'удалён')")

	roleVal, _ := c.Get("role")
	if roleVal == role.Physicist {
		db = db.Where("physicist_id = ?", h.getPhysicistID(c))
	}

	if status != "" {
		db = db.Where("status = ?", status)
	}
	if dateFrom != "" && dateTo != "" {
		db = db.Where("formed_at BETWEEN ? AND ?", dateFrom, dateTo)
	}

	db.Find(&calculations)

	var result []map[string]interface{}
	for _, req := range calculations {
		validItemsCount := 0
		hasPriority := false // <--- Флаг наличия срочных элементов

		for _, item := range req.Items {
			if item.CalculatedCurrent > 0 {
				validItemsCount++
			}
			if item.IsPriority { // Если хоть один Item срочный
				hasPriority = true
			}
		}

		result = append(result, map[string]interface{}{
			"id":                  req.ID,
			"theme":               req.Theme,   // Передаем тему для фронтенда
			"is_priority":         hasPriority, // Передаем флаг для красного бейджика в журнале!
			"status":              req.Status,
			"created_at":          req.CreatedAt,
			"formed_at":           req.FormedAt,
			"total_current":       req.TotalCurrent,
			"creator_login":       req.Physicist.Login,
			"valid_results_count": validItemsCount,
		})
	}
	c.JSON(http.StatusOK, result)
}

// @Summary Получение заявки по ID
func (h *Handler) GetCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	var req ds.RadiationCalculation
	if err := h.repo.GetDB().Preload("Items.Radiation").First(&req, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}
	c.JSON(http.StatusOK, req)
}

// Универсальная структура для обновления полей заявки
type UpdateCalculationRequest struct {
	Theme        *string  `json:"theme"`
	Description  *string  `json:"description"`
	TotalCurrent *float64 `json:"total_current"`
}

// @Summary Изменение полей заявки (тема, описание, итог)
func (h *Handler) UpdateCalculationAPI(c *gin.Context) {
	id := c.Param("id")

	var req UpdateCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	var calc ds.RadiationCalculation
	if err := h.repo.GetDB().First(&calc, "id = ? AND physicist_id = ?", id, h.getPhysicistID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	if calc.Status != "draft" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Изменять поля можно только у черновика"})
		return
	}

	if req.Theme != nil {
		calc.Theme = *req.Theme
	}
	if req.Description != nil {
		calc.Description = *req.Description
	}
	if req.TotalCurrent != nil {
		calc.TotalCurrent = *req.TotalCurrent
	}

	if err := h.repo.GetDB().Save(&calc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении"})
		return
	}

	c.JSON(http.StatusOK, calc)
}

// @Summary Сформировать заявку (запуск расчетов)
func (h *Handler) FormCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	var req ds.RadiationCalculation

	if err := h.repo.GetDB().Preload("Items.Radiation").First(&req, "id = ? AND status = 'draft' AND physicist_id = ?", id, h.getPhysicistID(c)).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Только ваш черновик можно сформировать"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "В заявке нет услуг (обязательное поле)"})
		return
	}

	totalCurrent := 0.0
	const h_plank = 6.626e-34
	const e_charge = 1.6e-19
	const P_density = 100.0

	for i := range req.Items {
		freq := req.Items[i].Frequency
		workFunc_eV := req.Items[i].WorkFunction

		if freq <= 0 {
			req.Items[i].CalculatedCurrent = -2
			h.repo.GetDB().Save(&req.Items[i])
			continue
		}

		E_photon_J := h_plank * freq
		workFunc_J := workFunc_eV * e_charge
		E_k_J := E_photon_J - workFunc_J

		if E_k_J <= 0 {
			req.Items[i].CalculatedCurrent = -1
			h.repo.GetDB().Save(&req.Items[i])
			continue
		}

		area_m2 := req.Items[i].Area * 1e-4
		power_W := P_density * area_m2
		N_photons := power_W / E_photon_J
		N_electrons := N_photons * (req.Items[i].Efficiency / 100.0)
		current_A := N_electrons * e_charge

		req.Items[i].CalculatedCurrent = current_A * 1000
		h.repo.GetDB().Save(&req.Items[i])

		totalCurrent += req.Items[i].CalculatedCurrent
	}

	now := time.Now()
	h.repo.GetDB().Model(&req).Updates(map[string]interface{}{
		"status":        "сформирован",
		"formed_at":     now,
		"total_current": totalCurrent,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Заявка сформирована", "total_current": totalCurrent})
}

// @Summary Завершение или отклонение заявки
func (h *Handler) CompleteCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req ds.RadiationCalculation
	if err := h.repo.GetDB().First(&req, "id = ? AND status = 'сформирован'", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заявка должна быть в статусе 'сформирован'"})
		return
	}

	status := "завершён"
	if input.Action == "reject" {
		status = "отклонён"
	}

	modID := h.getPhysicistID(c)
	now := time.Now()

	h.repo.GetDB().Model(&req).Updates(map[string]interface{}{
		"status":       status,
		"professor_id": modID,
		"completed_at": now,
	})
	c.JSON(http.StatusOK, gin.H{"new_status": status})
}

// @Summary Логическое удаление заявки
func (h *Handler) DeleteCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	h.repo.GetDB().Model(&ds.RadiationCalculation{}).Where("id = ? AND physicist_id = ?", id, h.getPhysicistID(c)).Update("status", "удалён")
	c.Status(http.StatusNoContent)
}

// === ДОМЕН ПОЛЬЗОВАТЕЛЬ ===

func hashPassword(pass string) string {
	h := sha1.New()
	h.Write([]byte(pass))
	return hex.EncodeToString(h.Sum(nil))
}

// @Summary Регистрация пользователя
func (h *Handler) RegisterAPI(c *gin.Context) {
	var input struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     int    `json:"role"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := ds.Physicist{
		Login:    input.Login,
		Password: hashPassword(input.Password),
		Role:     role.Role(input.Role),
	}

	if err := h.repo.GetDB().Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка регистрации"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// @Summary Авторизация (Login)
func (h *Handler) LoginAPI(c *gin.Context) {
	var input struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var physicist ds.Physicist
	if err := h.repo.GetDB().Where("login = ?", input.Login).First(&physicist).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не найден"})
		return
	}

	if physicist.Password != hashPassword(input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный пароль"})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &JWTClaims{
		PhysicistID: physicist.ID,
		Role:        physicist.Role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":    physicist.ID,
			"login": physicist.Login,
			"role":  physicist.Role,
		},
		"expires_in": expirationTime.Unix(),
	})
}

// @Summary Выход (Logout)
func (h *Handler) LogoutAPI(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, jwtPrefix) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Токен не передан"})
		return
	}

	tokenStr := authHeader[len(jwtPrefix):]

	token, _ := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	var expTime time.Duration = 24 * time.Hour
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		expTime = time.Until(time.Unix(claims.ExpiresAt, 0))
	}

	h.repo.GetRedis().Set(c.Request.Context(), "blacklist:"+tokenStr, true, expTime)

	c.JSON(http.StatusOK, gin.H{"message": "Вы успешно вышли из системы"})
}
