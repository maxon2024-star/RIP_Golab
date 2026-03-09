package handler

import (
	"RIP_Golab/internal/app/ds"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

// === ДОМЕН УСЛУГИ ===

func (h *Handler) GetServicesAPI(c *gin.Context) {
	search := c.Query("search")
	var services []ds.RadiationRange

	db := h.repo.GetDB().Where("is_delete = ?", false)
	if search != "" {
		db = db.Where("name ILIKE ?", "%"+search+"%")
	}
	db.Find(&services)

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": services})
}

func (h *Handler) GetServiceAPI(c *gin.Context) {
	id := c.Param("id")
	var service ds.RadiationRange
	if err := h.repo.GetDB().First(&service, "id = ? AND is_delete = false", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Услуга не найдена"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": service})
}

func (h *Handler) AddServiceAPI(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка формы"})
		return
	}

	service := ds.RadiationRange{
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		EnergyRange: c.PostForm("energy_range"),
		Frequency:   c.PostForm("frequency"),
	}

	// Сохранение в MinIO
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
			bucket := "physicsservice" // Убедись, что бакет называется именно так в настройках

			// РЕАЛЬНАЯ ОТПРАВКА В MINIO
			_, err = h.repo.GetMinio().PutObject(
				context.Background(),
				bucket,
				filename,
				file,
				header.Size,
				minio.PutObjectOptions{ContentType: contentType},
			)

			if err != nil {
				logrus.Errorf("Ошибка загрузки в MinIO: %v", err)
				return ""
			}

			return "http://localhost:9000/" + bucket + "/" + filename
		}
		return ""
	}

	service.ImageURL = uploadToMinio("image")
	service.VideoURL = uploadToMinio("video")

	h.repo.GetDB().Create(&service)
	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": service})
}

// === ДОМЕН М-М (ЗАЯВКИ-УСЛУГИ) ===

type M2MInput struct {
	RadiationID  uint    `json:"radiation_id" binding:"required"`
	Area         float64 `json:"area"`
	Efficiency   float64 `json:"efficiency"`
	WorkFunction float64 `json:"work_function"`
	Frequency    float64 `json:"frequency"`
}

func (h *Handler) AddToCartAPI(c *gin.Context) {
	var input M2MInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := CurrentUser()
	var draft ds.RadiationCalculation

	// Если нет черновика, создаем пустой (status='draft', created_at=now)
	err := h.repo.GetDB().Where("user_id = ? AND status = 'draft'", userID).First(&draft).Error
	if err != nil {
		draft = ds.RadiationCalculation{UserID: userID, Status: "draft"}
		h.repo.GetDB().Create(&draft)
	}

	item := ds.CalculationItem{
		CalculationID: draft.ID,
		RadiationID:   input.RadiationID,
		Area:          input.Area,
		Efficiency:    input.Efficiency,
		WorkFunction:  input.WorkFunction,
		Frequency:     input.Frequency,
	}
	h.repo.GetDB().Create(&item)
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Добавлено в заявку"})
}

func (h *Handler) UpdateCartAPI(c *gin.Context) {
	var input M2MInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var draft ds.RadiationCalculation
	if err := h.repo.GetDB().Where("user_id = ? AND status = 'draft'", CurrentUser()).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Черновик не найден"})
		return
	}

	// Изменение без PK M2M - ищем по связи Заявка+Услуга
	h.repo.GetDB().Model(&ds.CalculationItem{}).
		Where("calculation_id = ? AND radiation_id = ?", draft.ID, input.RadiationID).
		Updates(map[string]interface{}{
			"area":          input.Area,
			"efficiency":    input.Efficiency,
			"work_function": input.WorkFunction,
			"frequency":     input.Frequency,
		})

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) DeleteFromCartAPI(c *gin.Context) {
	radiationID := c.Query("radiation_id")

	var draft ds.RadiationCalculation
	if err := h.repo.GetDB().Where("user_id = ? AND status = 'draft'", CurrentUser()).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Черновик не найден"})
		return
	}

	// Удаление без PK M2M
	h.repo.GetDB().Where("calculation_id = ? AND radiation_id = ?", draft.ID, radiationID).Delete(&ds.CalculationItem{})
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// === ДОМЕН ЗАЯВКИ ===

func (h *Handler) GetCartIconAPI(c *gin.Context) {
	userID := CurrentUser()
	var draft ds.RadiationCalculation

	if err := h.repo.GetDB().Preload("Items").Where("user_id = ? AND status = 'draft'", userID).First(&draft).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"draft_id": nil, "count": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft_id": draft.ID, "count": len(draft.Items)})
}

func (h *Handler) GetRequestsAPI(c *gin.Context) {
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	status := c.Query("status")

	var requests []ds.RadiationCalculation

	// Исключаем удаленные и черновики
	db := h.repo.GetDB().Preload("User").Preload("Items").
		Where("status NOT IN ('draft', 'удалён')")

	if status != "" {
		db = db.Where("status = ?", status)
	}
	if dateFrom != "" && dateTo != "" {
		db = db.Where("formed_at BETWEEN ? AND ?", dateFrom, dateTo)
	}

	db.Find(&requests)

	// Добавляем вычисляемое поле (количество записей м-м с непустым результатом)
	var result []map[string]interface{}
	for _, req := range requests {
		validItemsCount := 0
		for _, item := range req.Items {
			if item.CalculatedCurrent > 0 { // Проверка "не пустого" результата вычислений
				validItemsCount++
			}
		}

		result = append(result, map[string]interface{}{
			"id":                  req.ID,
			"status":              req.Status,
			"created_at":          req.CreatedAt,
			"formed_at":           req.FormedAt,
			"total_current":       req.TotalCurrent,
			"creator_login":       req.User.Login,  // поле создателя через логин
			"valid_results_count": validItemsCount, // вычисляемое поле
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *Handler) GetRequestAPI(c *gin.Context) {
	id := c.Param("id")
	var req ds.RadiationCalculation
	if err := h.repo.GetDB().Preload("Items.Radiation").First(&req, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": req})
}

func (h *Handler) UpdateRequestAPI(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.repo.GetDB().Model(&ds.RadiationCalculation{}).Where("id = ?", id).Update("description", input.Description)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) FormRequestAPI(c *gin.Context) {
	id := c.Param("id")
	var req ds.RadiationCalculation

	if err := h.repo.GetDB().Preload("Items").First(&req, "id = ? AND status = 'draft'", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Только черновик можно сформировать"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "В заявке нет услуг (обязательное поле)"})
		return
	}

	// === БИЗНЕС-ЛОГИКА: Вычисление формулы физики перенесено сюда по ТЗ ===
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

		req.Items[i].KineticEnergy = E_k_J / e_charge

		if E_k_J <= 0 {
			req.Items[i].KineticEnergy = 0
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

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Заявка сформирована", "total_current": totalCurrent})
}

func (h *Handler) CompleteRequestAPI(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Action string `json:"action" binding:"required"` // "complete" or "reject"
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

	modID := CurrentModerator()
	now := time.Now()

	h.repo.GetDB().Model(&req).Updates(map[string]interface{}{
		"status":       status,
		"moderator_id": modID,
		"completed_at": now,
	})

	c.JSON(http.StatusOK, gin.H{"status": "success", "new_status": status})
}

func (h *Handler) DeleteRequestAPI(c *gin.Context) {
	id := c.Param("id")
	h.repo.GetDB().Model(&ds.RadiationCalculation{}).Where("id = ?", id).Update("status", "удалён")
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Заявка логически удалена"})
}

// === ДОМЕН ПОЛЬЗОВАТЕЛЬ ===

func (h *Handler) RegisterAPI(c *gin.Context) {
	// Создаем структуру специально для чтения входящего JSON
	var input struct {
		Login       string `json:"login" binding:"required"`
		Password    string `json:"password" binding:"required"`
		IsModerator bool   `json:"is_moderator"`
	}

	// Читаем JSON в структуру input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Перекладываем данные в модель БД
	user := ds.User{
		Login:       input.Login,
		Password:    input.Password,
		IsModerator: input.IsModerator,
	}

	// Сохраняем в базу
	if err := h.repo.GetDB().Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка регистрации"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": user})
}

func (h *Handler) LoginAPI(c *gin.Context) {
	// Заглушка для 4ой лабораторной
	c.JSON(http.StatusOK, gin.H{"status": "success", "token": "dummy_jwt_token_here"})
}

func (h *Handler) LogoutAPI(c *gin.Context) {
	// Заглушка для 4ой лабораторной
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Деавторизация успешна"})
}
