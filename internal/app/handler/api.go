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
// @Description Возвращает список всех излучений из каталога
// @Tags Услуги
// @Produce json
// @Param search query string false "Поиск по названию"
// @Success 200 {array} ds.RadiationRange
// @Router /api/radiations [get]
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
// @Tags Услуги
// @Produce json
// @Param id path int true "ID Излучения"
// @Success 200 {object} ds.RadiationRange
// @Router /api/radiations/{id} [get]
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
// @Description Доступно только Профессору
// @Tags Услуги
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Название"
// @Param short_description formData string false "Краткое описание"
// @Param description formData string false "Описание"
// @Param image formData file false "Изображение"
// @Param video formData file false "Видео"
// @Success 201 {object} ds.RadiationRange
// @Router /api/radiations [post]
func (h *Handler) AddRadiationAPI(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка формы"})
		return
	}

	radiation := ds.RadiationRange{
		Name:             c.PostForm("name"),
		ShortDescription: c.PostForm("short_description"), // Обработка нового поля
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
}

// @Summary Добавление излучения в корзину (черновик)
// @Tags Корзина
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body M2MInput true "Параметры эксперимента"
// @Success 201 {object} map[string]string
// @Router /api/calculation-items [post]
func (h *Handler) AddCalculationItemAPI(c *gin.Context) {
	var input M2MInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация: проверяем, что пользователь передал частоту и работу выхода
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
	}
	h.repo.GetDB().Create(&item)
	c.JSON(http.StatusCreated, gin.H{"message": "Излучение добавлено в расчет"})
}

// @Summary Обновление элемента в корзине
// @Tags Корзина
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body M2MInput true "Новые параметры"
// @Success 200 "Успешно"
// @Router /api/calculation-items [put]
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

	h.repo.GetDB().Model(&ds.CalculationItem{}).
		Where("calculation_id = ? AND radiation_id = ?", draft.ID, input.RadiationID).
		Updates(map[string]interface{}{
			"area":          input.Area,
			"efficiency":    input.Efficiency,
			"frequency":     input.Frequency,
			"work_function": input.WorkFunction,
		})
	c.Status(http.StatusOK)
}

// @Summary Удаление элемента из корзины
// @Tags Корзина
// @Security BearerAuth
// @Param radiation_id query int true "ID Излучения"
// @Success 204 "Успешно удалено"
// @Router /api/calculation-items [delete]
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
// @Tags Заявки
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/calculations/draft-summary [get]
func (h *Handler) GetDraftSummaryAPI(c *gin.Context) {
	physicistID := h.getPhysicistID(c)

	// Если пользователь не авторизован (нет ID), возвращаем пустую корзину (статус 200)
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
// @Tags Заявки
// @Security BearerAuth
// @Produce json
// @Param status query string false "Фильтр по статусу"
// @Param date_from query string false "Дата от (YYYY-MM-DD)"
// @Param date_to query string false "Дата до (YYYY-MM-DD)"
// @Success 200 {array} map[string]interface{}
// @Router /api/calculations [get]
func (h *Handler) GetCalculationsAPI(c *gin.Context) {
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	status := c.Query("status")

	var calculations []ds.RadiationCalculation
	db := h.repo.GetDB().Preload("Physicist").Preload("Items").Where("status NOT IN ('draft', 'удалён')")

	// Если обычный Физик - видит только свои
	// Если Профессор - видит все
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
		for _, item := range req.Items {
			if item.CalculatedCurrent > 0 {
				validItemsCount++
			}
		}
		result = append(result, map[string]interface{}{
			"id":                  req.ID,
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
// @Tags Заявки
// @Security BearerAuth
// @Param id path int true "ID Заявки"
// @Success 200 {object} ds.RadiationCalculation
// @Router /api/calculations/{id} [get]
func (h *Handler) GetCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	var req ds.RadiationCalculation
	if err := h.repo.GetDB().Preload("Items.Radiation").First(&req, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}
	c.JSON(http.StatusOK, req)
}

// @Summary Изменение описания заявки
// @Tags Заявки
// @Security BearerAuth
// @Param id path int true "ID Заявки"
// @Param input body map[string]string true "Описание"
// @Success 200 "Успешно"
// @Router /api/calculations/{id} [put]
func (h *Handler) UpdateCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.repo.GetDB().Model(&ds.RadiationCalculation{}).Where("id = ? AND physicist_id = ?", id, h.getPhysicistID(c)).Update("description", input.Description)
	c.Status(http.StatusOK)
}

// @Summary Сформировать заявку (запуск расчетов)
// @Description Переводит заявку из draft в сформирован и считает формулы
// @Tags Заявки
// @Security BearerAuth
// @Param id path int true "ID Заявки"
// @Success 200 {object} map[string]interface{}
// @Router /api/calculations/{id}/form [put]
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
// @Description Доступно только Профессору
// @Tags Заявки
// @Security BearerAuth
// @Param id path int true "ID Заявки"
// @Param input body map[string]string true "Action: accept / reject"
// @Success 200 {object} map[string]string
// @Router /api/calculations/{id}/complete [put]
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
// @Tags Заявки
// @Security BearerAuth
// @Param id path int true "ID Заявки"
// @Success 204 "Успешно"
// @Router /api/calculations/{id} [delete]
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
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param input body map[string]interface{} true "Данные регистрации"
// @Success 201 {object} ds.Physicist
// @Router /api/users/register [post]
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
// @Description Выдает JWT токен в случае успешной авторизации
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param input body map[string]string true "Креды (login, password)"
// @Success 200 {object} map[string]interface{}
// @Router /api/users/login [post]
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

	// Генерируем JWT
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
		"access_token": tokenString,
		"expires_in":   expirationTime.Unix(),
		"role":         physicist.Role,
	})
}

// @Summary Выход (Logout)
// @Description Помещает переданный JWT в Blacklist Redis'а
// @Tags Пользователи
// @Security BearerAuth
// @Router /api/users/logout [post]
func (h *Handler) LogoutAPI(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, jwtPrefix) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Токен не передан"})
		return
	}

	tokenStr := authHeader[len(jwtPrefix):]

	// Парсим токен, чтобы узнать его время жизни
	token, _ := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	var expTime time.Duration = 24 * time.Hour
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// Оставляем в Redis ровно до истечения срока действия самого токена
		expTime = time.Until(time.Unix(claims.ExpiresAt, 0))
	}

	// Записываем токен в Blacklist Redis
	h.repo.GetRedis().Set(c.Request.Context(), "blacklist:"+tokenStr, true, expTime)

	c.JSON(http.StatusOK, gin.H{"message": "Вы успешно вышли из системы"})
}
