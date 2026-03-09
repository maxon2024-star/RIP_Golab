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

// Функция-singleton пользователя по ТЗ
func CurrentUser() uint {
	return 1 // Константа: всегда работаем от лица пользователя ID=1 (user)
}
func CurrentModerator() uint {
	return 2 // Константа: модератор ID=2 (admin)
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/", h.GetServiceList)
	router.GET("/service/:id", h.GetServiceDetail)
	router.GET("/radiation_calculation/:id", h.GetCalculationByID)

	router.POST("/add-to-calculation", h.AddToCalculation)
	router.POST("/delete-calculation/:id", h.DeleteCalculation)

	router.POST("/delete-item", h.DeleteItem)
	router.POST("/update-item", h.UpdateItem)

	router.POST("/status-calculation", h.StatusCalculation)

	// ---------------- НОВЫЕ REST API РОУТЫ (/api) ----------------
	api := router.Group("/api")
	{
		// Домен услуги
		api.GET("/services", h.GetServicesAPI)
		api.GET("/services/:id", h.GetServiceAPI)
		api.POST("/services", h.AddServiceAPI) // Multipart form

		// Домен м-м (Корзина/Услуги заявки)
		api.POST("/cart", h.AddToCartAPI)
		api.PUT("/cart", h.UpdateCartAPI)
		api.DELETE("/cart", h.DeleteFromCartAPI)

		// Домен заявки
		api.GET("/cart/icon", h.GetCartIconAPI)
		api.GET("/requests", h.GetRequestsAPI)
		api.GET("/requests/:id", h.GetRequestAPI)
		api.PUT("/requests/:id", h.UpdateRequestAPI)
		api.PUT("/requests/:id/form", h.FormRequestAPI)         // Вычисление формулы тут!
		api.PUT("/requests/:id/complete", h.CompleteRequestAPI) // Завершить/Отклонить
		api.DELETE("/requests/:id", h.DeleteRequestAPI)

		// Домен пользователь
		api.POST("/register", h.RegisterAPI)
		api.POST("/login", h.LoginAPI)
		api.POST("/logout", h.LogoutAPI)
	}
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

	// ФИЗИЧЕСКИЕ КОНСТАНТЫ СИ
	const h_plank = 6.626e-34 // Постоянная Планка (Дж·с)
	const e_charge = 1.6e-19  // Заряд электрона (Кл)
	const P_density = 100.0   // Интенсивность (мощность) падающего света Вт/м^2 (Константа)

	for i := range request.Items {
		freq := request.Items[i].Frequency
		workFunc_eV := request.Items[i].WorkFunction

		// Ошибка 1: Пользователь ввел 0 или отрицательную частоту
		if freq <= 0 {
			request.Items[i].CalculatedCurrent = -2
			h.repo.GetDB().Save(&request.Items[i])
			continue
		}

		// 1. Энергия падающего фотона E = h * v (в Джоулях)
		E_photon_J := h_plank * freq

		// Переводим работу выхода из эВ в Джоули
		workFunc_J := workFunc_eV * e_charge

		// 2. Уравнение Эйнштейна: Кинетическая энергия E_k = h*v - A
		E_k_J := E_photon_J - workFunc_J

		// Сохраняем кинетическую энергию в эВ для отображения в таблице
		request.Items[i].KineticEnergy = E_k_J / e_charge

		// Ошибка 2: Красная граница фотоэффекта. Энергии не хватает (h*v < A)
		if E_k_J <= 0 {
			request.Items[i].KineticEnergy = 0
			request.Items[i].CalculatedCurrent = -1
			h.repo.GetDB().Save(&request.Items[i])
			continue
		}

		// 3. Расчет тока насыщения (I = N_e * e)
		area_m2 := request.Items[i].Area * 1e-4                          // Площадь в м^2
		power_W := P_density * area_m2                                   // Мощность света на эту площадь (Вт = Дж/с)
		N_photons := power_W / E_photon_J                                // Кол-во падающих фотонов в секунду
		N_electrons := N_photons * (request.Items[i].Efficiency / 100.0) // Выбитые электроны с учетом КПД
		current_A := N_electrons * e_charge                              // Ток в Амперах

		// Перевод в мА
		request.Items[i].CalculatedCurrent = current_A * 1000
		h.repo.GetDB().Save(&request.Items[i])

		totalCurrent += request.Items[i].CalculatedCurrent
	}

	h.repo.GetDB().Model(request).Update("total_current", totalCurrent)

	c.HTML(http.StatusOK, "request.html", gin.H{
		"request":      request,
		"count":        h.repo.GetCalculationItemCount(h.getUserID(c)),
		"totalCurrent": totalCurrent,
	})
}

func (h *Handler) AddToCalculation(c *gin.Context) {
	serviceID, _ := strconv.Atoi(c.PostForm("service_id"))
	area, _ := strconv.ParseFloat(c.PostForm("area"), 64)
	if area == 0 {
		area = 10
	}
	efficiency, _ := strconv.ParseFloat(c.PostForm("efficiency"), 64)
	if efficiency == 0 {
		efficiency = 18
	}
	workFunc, _ := strconv.ParseFloat(c.PostForm("work_function"), 64)
	if workFunc == 0 {
		workFunc = 2.0
	} // Дефолт работы выхода (например, Цезий)

	// БЕРЕМ ЧАСТОТУ КАК ЕСТЬ. ЕСЛИ ПУСТАЯ - БУДЕТ 0 (Выдаст ошибку пользователю!)
	frequency, _ := strconv.ParseFloat(c.PostForm("frequency"), 64)

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
			WorkFunction:  workFunc,
			Frequency:     frequency,
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
	workFunc, _ := strconv.ParseFloat(c.PostForm("work_function"), 64)
	frequency, _ := strconv.ParseFloat(c.PostForm("frequency"), 64)

	h.repo.GetDB().Model(&ds.CalculationItem{}).Where("id = ?", itemID).Updates(map[string]interface{}{
		"area":          area,
		"efficiency":    efficiency,
		"work_function": workFunc,
		"frequency":     frequency,
	})
	c.Redirect(http.StatusFound, "/radiation_calculation/"+calcIDStr)
}

// ОТПРАВКА НА МОДЕРАЦИЮ
func (h *Handler) StatusCalculation(c *gin.Context) {
	calcIDStr := c.PostForm("calc_id")
	calcID, err := strconv.Atoi(calcIDStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// Обновляем статус заявки на "сформирован" и фиксируем время
	h.repo.GetDB().Exec("UPDATE radiation_calculations SET status = 'сформирован', formed_at = NOW() WHERE id = ?", calcID)

	// После отправки кидаем пользователя на главную страницу
	c.Redirect(http.StatusFound, "/")
}
