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

func CurrentUser() uint      { return 1 }
func CurrentModerator() uint { return 2 }

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/", h.GetServiceList)
	router.GET("/service/:id", h.GetServiceDetail)
	router.GET("/radiation_calculation/:id", h.GetCalculationByID)

	router.POST("/add-to-calculation", h.AddToCalculation)
	router.POST("/delete-calculation/:id", h.DeleteCalculation)

	router.POST("/delete-item", h.DeleteItem)
	router.POST("/update-item", h.UpdateItem)

	router.POST("/status-calculation", h.StatusCalculation)

	api := router.Group("/api")
	{
		api.GET("/radiations", h.GetRadiationsAPI)
		api.GET("/radiations/:id", h.GetRadiationAPI)
		api.POST("/radiations", h.AddRadiationAPI)

		api.POST("/calculation-items", h.AddCalculationItemAPI)
		api.PUT("/calculation-items", h.UpdateCalculationItemAPI)
		api.DELETE("/calculation-items", h.DeleteCalculationItemAPI)

		api.GET("/calculations/draft-summary", h.GetDraftSummaryAPI)
		api.GET("/calculations", h.GetCalculationsAPI)
		api.GET("/calculations/:id", h.GetCalculationAPI)
		api.PUT("/calculations/:id", h.UpdateCalculationAPI)
		api.PUT("/calculations/:id/form", h.FormCalculationAPI)
		api.PUT("/calculations/:id/complete", h.CompleteCalculationAPI)
		api.DELETE("/calculations/:id", h.DeleteCalculationAPI)

		api.POST("/users/register", h.RegisterAPI)
		api.POST("/users/login", h.LoginAPI)
		api.POST("/users/logout", h.LogoutAPI)
	}
}

func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./resources/static")
	router.Static("/img", "./resources/img")
}

func (h *Handler) errorHandler(ctx *gin.Context, statusCode int, err error) {
	logrus.Error(err.Error())
	ctx.Redirect(http.StatusFound, "/")
}

func (h *Handler) getUserID(c *gin.Context) uint {
	return CurrentUser()
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
	const h_plank = 6.626e-34
	const e_charge = 1.6e-19
	const P_density = 100.0

	for i := range request.Items {
		// ТЕПЕРЬ МЫ БЕРЕМ ЗНАЧЕНИЯ ПРЯМО ИЗ БАЗЫ ДАННЫХ, А НЕ ИЗ КАТАЛОГА!
		freq := request.Items[i].Frequency
		workFunc_eV := request.Items[i].WorkFunction

		if freq <= 0 {
			request.Items[i].CalculatedCurrent = -2
			h.repo.GetDB().Save(&request.Items[i])
			continue
		}

		E_photon_J := h_plank * freq
		workFunc_J := workFunc_eV * e_charge
		E_k_J := E_photon_J - workFunc_J

		request.Items[i].KineticEnergy = E_k_J / e_charge

		if E_k_J <= 0 {
			request.Items[i].KineticEnergy = 0
			request.Items[i].CalculatedCurrent = -1
			h.repo.GetDB().Save(&request.Items[i])
			continue
		}

		area_m2 := request.Items[i].Area * 1e-4
		power_W := P_density * area_m2
		N_photons := power_W / E_photon_J
		N_electrons := N_photons * (request.Items[i].Efficiency / 100.0)
		current_A := N_electrons * e_charge

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

	areaStr := strings.ReplaceAll(c.PostForm("area"), ",", ".")
	effStr := strings.ReplaceAll(c.PostForm("efficiency"), ",", ".")
	freqStr := strings.ReplaceAll(c.PostForm("frequency"), ",", ".")
	workFuncStr := strings.ReplaceAll(c.PostForm("work_function"), ",", ".")

	area, _ := strconv.ParseFloat(areaStr, 64)
	if area == 0 {
		area = 10
	}

	efficiency, _ := strconv.ParseFloat(effStr, 64)
	if efficiency == 0 {
		efficiency = 18
	}

	frequency, _ := strconv.ParseFloat(freqStr, 64)
	workFunc, _ := strconv.ParseFloat(workFuncStr, 64)

	// Берем дефолты из справочника, только если чувак оставил поля пустыми (при создании)
	var rad ds.RadiationRange
	h.repo.GetDB().First(&rad, serviceID)

	if frequency == 0 {
		frequency = parseFrequency(rad.Frequency)
	}
	if workFunc == 0 {
		workFunc = rad.WorkFunction
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
			Frequency:     frequency,
			WorkFunction:  workFunc,
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
		"кГц": 1e3, "МГц": 1e6, "ГГц": 1e9, "ТГц": 1e12, "ПГц": 1e15, "ЭГц": 1e18,
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

func (h *Handler) DeleteItem(c *gin.Context) {
	itemID, _ := strconv.Atoi(c.PostForm("item_id"))
	calcIDStr := c.PostForm("calc_id")
	h.repo.GetDB().Delete(&ds.CalculationItem{}, itemID)
	c.Redirect(http.StatusFound, "/radiation_calculation/"+calcIDStr)
}

func (h *Handler) UpdateItem(c *gin.Context) {
	itemID, _ := strconv.Atoi(c.PostForm("item_id"))
	calcIDStr := c.PostForm("calc_id")

	areaStr := strings.ReplaceAll(c.PostForm("area"), ",", ".")
	effStr := strings.ReplaceAll(c.PostForm("efficiency"), ",", ".")
	freqStr := strings.ReplaceAll(c.PostForm("frequency"), ",", ".")
	workFuncStr := strings.ReplaceAll(c.PostForm("work_function"), ",", ".")

	area, _ := strconv.ParseFloat(areaStr, 64)
	efficiency, _ := strconv.ParseFloat(effStr, 64)
	frequency, _ := strconv.ParseFloat(freqStr, 64)
	workFunc, _ := strconv.ParseFloat(workFuncStr, 64)

	// СОХРАНЯЕМ В БАЗУ ВСЕ 4 ПОЛЯ!
	h.repo.GetDB().Model(&ds.CalculationItem{}).Where("id = ?", itemID).Updates(map[string]interface{}{
		"area":          area,
		"efficiency":    efficiency,
		"frequency":     frequency,
		"work_function": workFunc,
	})

	c.Redirect(http.StatusFound, "/radiation_calculation/"+calcIDStr)
}

func (h *Handler) StatusCalculation(c *gin.Context) {
	calcIDStr := c.PostForm("calc_id")
	calcID, err := strconv.Atoi(calcIDStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	h.repo.GetDB().Exec("UPDATE radiation_calculations SET status = 'сформирован', formed_at = NOW() WHERE id = ?", calcID)

	c.Redirect(http.StatusFound, "/")
}
