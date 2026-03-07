package handler

import (
	"RIP_Golab/internal/app/ds"
	"RIP_Golab/internal/app/repository"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	repo *repository.Repository
}

func NewHandler(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/", h.GetServiceList)
	router.GET("/service/:id", h.GetServiceDetail)
	router.GET("/radiation_calculation/:id", h.GetCalculationByID)

	router.POST("/add-to-calculation", h.AddToCalculation)
	router.POST("/delete-calculation/:id", h.DeleteCalculation)

	router.POST("/delete-item", h.DeleteItem)
	router.POST("/update-item", h.UpdateItem)
}
func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./resources/static")
	router.Static("/img", "./resources/img")
}

// ИСПРАВЛЕНИЕ: Вместо 404.html делаем редирект
func (h *Handler) errorHandler(ctx *gin.Context, statusCode int, err error) {
	logrus.Error(err.Error())
	ctx.Redirect(http.StatusFound, "/")
}

func (h *Handler) getUserID(c *gin.Context) uint {
	return 1
}

func (h *Handler) GetServiceList(c *gin.Context) {
	search := c.Query("search")
	var services []ds.RadiationRange
	var err error

	if search != "" {
		services, err = h.repo.SearchRadiationRangesByName(search)
	} else {
		services, err = h.repo.GetAllRadiationRanges()
	}

	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	userID := h.getUserID(c)
	count := h.repo.GetCalculationItemCount(userID)

	var draftID uint = 0
	draft, err := h.repo.GetCalculationByUserID(userID)
	if err == nil && draft != nil {
		draftID = draft.ID
	}

	c.HTML(http.StatusOK, "services.html", gin.H{
		"services": services,
		"count":    count,
		"search":   search,
		"draft_id": draftID,
	})
}

func (h *Handler) GetServiceDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	service, err := h.repo.GetRadiationRangeByID(uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	videoURL := service.VideoURL
	if videoURL == "" {
		videoURL = service.ImageURL
		videoURL = strings.Replace(videoURL, ".jpg", ".mp4", 1)
		videoURL = strings.Replace(videoURL, ".png", ".mp4", 1)
	}

	userID := h.getUserID(c)
	count := h.repo.GetCalculationItemCount(userID)

	var draftID uint = 0
	draft, err := h.repo.GetCalculationByUserID(userID)
	if err == nil && draft != nil {
		draftID = draft.ID
	}

	c.HTML(http.StatusOK, "service_detail.html", gin.H{
		"service":  service,
		"videoURL": videoURL,
		"count":    count,
		"draft_id": draftID,
	})
}

func (h *Handler) GetCalculationByID(c *gin.Context) {
	idParam := c.Param("id")
	id, _ := strconv.Atoi(idParam)

	request, err := h.repo.GetCalculationByID(uint(id))
	if err != nil || request.Status == "удалён" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	totalCurrent := 0.0
	for i := range request.Items {
		// Логика получения числовой частоты
		var freqValue float64

		// 1. Пытаемся взять частоту из самого айтема (уже в Гц)
		freqValue = request.Items[i].Frequency

		// 2. Если в айтеме 0, пробуем распарсить строку из RadiationRange
		if freqValue == 0 {
			freqValue = parseFrequency(request.Items[i].Radiation.Frequency)
		}

		// --- ФОРМУЛА РАСЧЕТА ---
		// Ток (мА) = Площадь * КПД * (Частота / 10^14) / 1000
		// Коэффициент 1e14 выбран как средний для видимого света
		frequencyFactor := freqValue / 1e14

		// Ограничители, чтобы значения не улетали в бесконечность
		if frequencyFactor > 100 {
			frequencyFactor = 100
		}
		if frequencyFactor < 0.0001 {
			frequencyFactor = 0.0001
		}

		current := request.Items[i].Area * (request.Items[i].Efficiency / 100) * frequencyFactor
		request.Items[i].CalculatedCurrent = current

		// Сохраняем результат в БД
		h.repo.GetDB().Model(&request.Items[i]).Update("calculated_current", current)

		totalCurrent += current
	}

	// Обновляем общий итог в заявке
	h.repo.GetDB().Model(request).Update("total_current", totalCurrent)

	c.HTML(http.StatusOK, "request.html", gin.H{
		"request":      request,
		"count":        h.repo.GetCalculationItemCount(h.getUserID(c)),
		"totalCurrent": totalCurrent,
	})
}

func (h *Handler) AddToCalculation(c *gin.Context) {
	serviceIDStr := c.PostForm("service_id")
	areaStr := c.PostForm("area")
	efficiencyStr := c.PostForm("efficiency")
	frequencyStr := c.PostForm("frequency") // Приходит как строка

	serviceID, _ := strconv.Atoi(serviceIDStr)
	area, _ := strconv.ParseFloat(areaStr, 64)
	if area == 0 {
		area = 10
	}
	efficiency, _ := strconv.ParseFloat(efficiencyStr, 64)
	if efficiency == 0 {
		efficiency = 18
	}

	var frequency float64
	// Если пользователь оставил поле частоты пустым - берем стандартную из услуги
	if frequencyStr == "" {
		service, _ := h.repo.GetRadiationRangeByID(uint(serviceID))
		if service != nil {
			frequency = parseFrequency(service.Frequency) // преобразуем строку из БД в число Гц
		}
	} else {
		frequency, _ = strconv.ParseFloat(frequencyStr, 64)
	}

	userID := h.getUserID(c)

	request, err := h.repo.GetCalculationByUserID(userID)
	if err != nil {
		request = &ds.RadiationCalculation{UserID: userID, Status: "draft"}
		h.repo.CreateCalculation(request)
	}

	var existingCount int64
	h.repo.GetDB().Model(&ds.CalculationItem{}).
		Where("calculation_id = ? AND radiation_id = ?", request.ID, uint(serviceID)).
		Count(&existingCount)

	if existingCount == 0 {
		item := &ds.CalculationItem{
			CalculationID: request.ID,
			RadiationID:   uint(serviceID),
			Area:          area,
			Efficiency:    efficiency,
			Frequency:     frequency, // Теперь сохраняем float64
		}
		h.repo.AddItemToCalculation(item)
	}

	c.Redirect(http.StatusFound, "/radiation_calculation/"+strconv.Itoa(int(request.ID)))
}

func (h *Handler) DeleteCalculation(c *gin.Context) {
	idParam := c.Param("id")
	id, _ := strconv.Atoi(idParam)
	h.repo.DeleteCalculationSQL(uint(id))
	c.Redirect(http.StatusFound, "/")
}

func parseFrequency(freqStr string) float64 {
	if freqStr == "" {
		return 0
	}
	multipliers := map[string]float64{
		"кГц": 1e3, "МГц": 1e6, "ГГц": 1e9, "ТГц": 1e12, "ПГц": 1e15,
	}
	for unit, mult := range multipliers {
		if strings.Contains(freqStr, unit) {
			parts := strings.Split(freqStr, " ")
			for _, part := range parts {
				if strings.Contains(part, unit) {
					numStr := strings.Replace(part, unit, "", 1)
					numStr = strings.TrimSpace(numStr)

					if strings.Contains(freqStr, "-") {
						parts := strings.Split(freqStr, "-")
						if len(parts) == 2 {
							numStr = strings.TrimSpace(parts[1])
							numStr = strings.Replace(numStr, unit, "", 1)
						}
					}
					num, _ := strconv.ParseFloat(numStr, 64)
					return num * mult
				}
			}
		}
	}
	return 0
}

// УДАЛЕНИЕ ИЗ КОРЗИНЫ
func (h *Handler) DeleteItem(c *gin.Context) {
	itemID, _ := strconv.Atoi(c.PostForm("item_id"))
	calcIDStr := c.PostForm("calc_id")
	h.repo.GetDB().Delete(&ds.CalculationItem{}, itemID)
	c.Redirect(http.StatusFound, "/radiation_calculation/"+calcIDStr)
}

// ОБНОВЛЕНИЕ ЗНАЧЕНИЙ В КОРЗИНЕ
func (h *Handler) UpdateItem(c *gin.Context) {
	itemID, _ := strconv.Atoi(c.PostForm("item_id"))
	calcIDStr := c.PostForm("calc_id")
	area, _ := strconv.ParseFloat(c.PostForm("area"), 64)
	efficiency, _ := strconv.ParseFloat(c.PostForm("efficiency"), 64)
	frequency, _ := strconv.ParseFloat(c.PostForm("frequency"), 64) // Парсим число

	h.repo.GetDB().Model(&ds.CalculationItem{}).Where("id = ?", itemID).Updates(map[string]interface{}{
		"area":       area,
		"efficiency": efficiency,
		"frequency":  frequency,
	})
	c.Redirect(http.StatusFound, "/radiation_calculation/"+calcIDStr)
}
