package handler

import (
	"RIP_Golab/internal/app/ds"
	"RIP_Golab/internal/app/repository"
	"RIP_Golab/internal/app/role"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handler struct {
	repo *repository.Repository
}

func NewHandler(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.Use(CORSMiddleware())

	router.GET("/", h.GetServiceList)
	router.GET("/service/:id", h.GetServiceDetail)
	router.GET("/radiation_calculation/:id", h.GetCalculationByID)

	router.POST("/add-to-calculation", h.AddToCalculation)
	router.POST("/delete-calculation/:id", h.DeleteCalculation)

	router.POST("/delete-item", h.DeleteItem)
	router.POST("/update-item", h.UpdateItem)

	router.POST("/status-calculation", h.StatusCalculation)

	// Swagger Route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api")
	{
		// Общедоступные (Guest)
		api.GET("/radiations", h.GetRadiationsAPI)
		api.GET("/radiations/:id", h.GetRadiationAPI)
		api.POST("/users/register", h.RegisterAPI)
		api.POST("/users/login", h.LoginAPI)

		// Доступ для Физиков (авторизованные пользователи)
		authGroup := api.Group("/")
		authGroup.Use(h.WithAuthCheck(role.Physicist))
		{
			authGroup.POST("/users/logout", h.LogoutAPI)
			authGroup.POST("/calculation-items", h.AddCalculationItemAPI)
			authGroup.PUT("/calculation-items", h.UpdateCalculationItemAPI)
			authGroup.DELETE("/calculation-items", h.DeleteCalculationItemAPI)

			authGroup.GET("/calculations/draft-summary", h.GetDraftSummaryAPI)
			authGroup.GET("/calculations", h.GetCalculationsAPI)
			authGroup.GET("/calculations/:id", h.GetCalculationAPI)
			authGroup.PUT("/calculations/:id", h.UpdateCalculationAPI)
			authGroup.PUT("/calculations/:id/form", h.FormCalculationAPI)
			authGroup.DELETE("/calculations/:id", h.DeleteCalculationAPI)
		}

		// Доступ только для Профессоров (модераторы)
		profGroup := api.Group("/")
		profGroup.Use(h.WithAuthCheck(role.Professor))
		{
			profGroup.POST("/radiations", h.AddRadiationAPI)
			profGroup.PUT("/calculations/:id/complete", h.CompleteCalculationAPI)
		}
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

// Для старых HTML-шаблонов возвращаем ID физика, если он есть, иначе отдаем 1 (заглушка)
func (h *Handler) getHTMLPhysicistID(c *gin.Context) uint {
	id := h.getPhysicistID(c)
	if id == 0 {
		return 1 // Хардкод fallback для старых HTML-страниц без JWT
	}
	return id
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

	physicistID := h.getHTMLPhysicistID(c)
	count := h.repo.GetCalculationItemCount(physicistID)

	var draftID uint = 0
	draft, err := h.repo.GetCalculationByPhysicistID(physicistID)
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

	physicistID := h.getHTMLPhysicistID(c)
	count := h.repo.GetCalculationItemCount(physicistID)

	var draftID uint = 0
	draft, err := h.repo.GetCalculationByPhysicistID(physicistID)
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
		"count":        h.repo.GetCalculationItemCount(h.getHTMLPhysicistID(c)),
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

	// Подставляем дефолтные значения для старых HTML-форм, если юзер не ввел параметры
	if frequency == 0 {
		frequency = 1e15
	}
	if workFunc == 0 {
		workFunc = 4.5
	}

	physicistID := h.getHTMLPhysicistID(c)
	request, err := h.repo.GetCalculationByPhysicistID(physicistID)
	if err != nil {
		request = &ds.RadiationCalculation{PhysicistID: physicistID, Status: "draft"}
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

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Разрешаем запросы с любых адресов
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		// Разрешаем нужные заголовки
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		// Разрешаем методы, включая OPTIONS!
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		// Если это предварительный запрос браузера (OPTIONS) - просто отвечаем 204 и не пускаем дальше
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
