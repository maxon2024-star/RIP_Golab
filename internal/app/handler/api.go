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

// === ДОМЕН УСЛУГИ (ИЗЛУЧЕНИЯ) ===

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

func (h *Handler) GetRadiationAPI(c *gin.Context) {
	id := c.Param("id")
	var radiation ds.RadiationRange
	if err := h.repo.GetDB().First(&radiation, "id = ? AND is_delete = false", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Излучение не найдено"})
		return
	}
	c.JSON(http.StatusOK, radiation)
}

func (h *Handler) AddRadiationAPI(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка формы"})
		return
	}

	radiation := ds.RadiationRange{
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		EnergyRange: c.PostForm("energy_range"),
		Frequency:   c.PostForm("frequency"),
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

func (h *Handler) AddCalculationItemAPI(c *gin.Context) {
	var input M2MInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Подгружаем дефолты из справочника, если через API передали 0
	var rad ds.RadiationRange
	h.repo.GetDB().First(&rad, input.RadiationID)

	if input.Frequency == 0 {
		input.Frequency = parseFrequencyStr(rad.Frequency)
	}
	if input.WorkFunction == 0 {
		input.WorkFunction = rad.WorkFunction
	}

	userID := CurrentUser()
	var draft ds.RadiationCalculation

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
		Frequency:     input.Frequency,
		WorkFunction:  input.WorkFunction,
	}
	h.repo.GetDB().Create(&item)
	c.JSON(http.StatusCreated, gin.H{"message": "Излучение добавлено в расчет"})
}

func (h *Handler) UpdateCalculationItemAPI(c *gin.Context) {
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

	// Обновляем все 4 поля
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

func (h *Handler) DeleteCalculationItemAPI(c *gin.Context) {
	radiationID := c.Query("radiation_id")

	var draft ds.RadiationCalculation
	if err := h.repo.GetDB().Where("user_id = ? AND status = 'draft'", CurrentUser()).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Черновик не найден"})
		return
	}

	h.repo.GetDB().Where("calculation_id = ? AND radiation_id = ?", draft.ID, radiationID).Delete(&ds.CalculationItem{})
	c.Status(http.StatusNoContent)
}

// === ДОМЕН ЗАЯВКИ (РАСЧЕТЫ) ===

func (h *Handler) GetDraftSummaryAPI(c *gin.Context) {
	userID := CurrentUser()
	var draft ds.RadiationCalculation

	if err := h.repo.GetDB().Preload("Items").Where("user_id = ? AND status = 'draft'", userID).First(&draft).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"draft_id": nil, "count": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft_id": draft.ID, "count": len(draft.Items)})
}

func (h *Handler) GetCalculationsAPI(c *gin.Context) {
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	status := c.Query("status")

	var calculations []ds.RadiationCalculation

	db := h.repo.GetDB().Preload("User").Preload("Items").
		Where("status NOT IN ('draft', 'удалён')")

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
			"creator_login":       req.User.Login,
			"valid_results_count": validItemsCount,
		})
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	var req ds.RadiationCalculation
	if err := h.repo.GetDB().Preload("Items.Radiation").First(&req, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) UpdateCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.repo.GetDB().Model(&ds.RadiationCalculation{}).Where("id = ?", id).Update("description", input.Description)
	c.Status(http.StatusOK)
}

func parseFrequencyStr(freqStr string) float64 {
	if freqStr == "" {
		return 0
	}
	multipliers := map[string]float64{
		"кГц": 1e3, "МГц": 1e6, "ГГц": 1e9, "ТГц": 1e12, "ПГц": 1e15, "ЭГц": 1e18,
	}
	for unit, mult := range multipliers {
		if strings.Contains(freqStr, unit) {
			parts := strings.Split(freqStr, "-")
			targetPart := parts[0]
			if len(parts) > 1 {
				targetPart = parts[1]
			}
			numStr := strings.TrimSpace(strings.Replace(targetPart, unit, "", 1))
			numStr = strings.Split(numStr, " ")[0]

			var num float64
			fmt.Sscanf(numStr, "%f", &num)
			return num * mult
		}
	}
	return 0
}

func (h *Handler) FormCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	var req ds.RadiationCalculation

	if err := h.repo.GetDB().Preload("Items.Radiation").First(&req, "id = ? AND status = 'draft'", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Только черновик можно сформировать"})
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
		// ТЕПЕРЬ БЕРЕМ ПАРАМЕТРЫ ИЗ М-М ТАБЛИЦЫ
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

	modID := CurrentModerator()
	now := time.Now()

	h.repo.GetDB().Model(&req).Updates(map[string]interface{}{
		"status":       status,
		"moderator_id": modID,
		"completed_at": now,
	})

	c.JSON(http.StatusOK, gin.H{"new_status": status})
}

func (h *Handler) DeleteCalculationAPI(c *gin.Context) {
	id := c.Param("id")
	h.repo.GetDB().Model(&ds.RadiationCalculation{}).Where("id = ?", id).Update("status", "удалён")
	c.Status(http.StatusNoContent)
}

// === ДОМЕН ПОЛЬЗОВАТЕЛЬ ===

func (h *Handler) RegisterAPI(c *gin.Context) {
	var input struct {
		Login       string `json:"login" binding:"required"`
		Password    string `json:"password" binding:"required"`
		IsModerator bool   `json:"is_moderator"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := ds.User{
		Login:       input.Login,
		Password:    input.Password,
		IsModerator: input.IsModerator,
	}

	if err := h.repo.GetDB().Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка регистрации"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) LoginAPI(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"token": "dummy_jwt_token_here"})
}

func (h *Handler) LogoutAPI(c *gin.Context) {
	c.Status(http.StatusOK)
}
