package handler

import (
	"RIP_Golab/internal/app/ds"
	"RIP_Golab/internal/app/repository"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	repo *repository.Repository
}

func NewHandler(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

type InputAddToRequest struct {
	ServiceID    int     `json:"service_id" binding:"required"`
	Area         float64 `json:"area"`
	WorkFunction float64 `json:"work_function"`
	Efficiency   float64 `json:"efficiency"`
	Intensity    float64 `json:"intensity"`
	Comment      string  `json:"comment"`
	Order        int     `json:"order"`
}

type InputUpdateRequest struct {
	ID           uint    `json:"id"`
	RequestID    uint    `json:"request_id"`
	RadiationID  uint    `json:"radiation_id"`
	ServiceID    uint    `json:"service_id"`
	Area         float64 `json:"area"`
	WorkFunction float64 `json:"work_function"`
	Efficiency   float64 `json:"efficiency"`
	Intensity    float64 `json:"intensity"`
	Comment      string  `json:"comment"`
	Order        int     `json:"order"`
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.POST("/register", h.Register)
	router.POST("/login", h.Login)
	router.POST("/logout", h.Logout)
	router.GET("/user", h.GetUser)

	router.GET("/", h.GetServiceList)
	router.GET("/service/:id", h.GetServiceDetail)

	router.GET("/request", h.GetRequest)
	router.POST("/add-to-request", h.AddToRequest)
	router.POST("/update-request", h.UpdateRequest)
	router.POST("/remove-from-request", h.RemoveFromRequest)
	router.POST("/delete-request", h.DeleteRequest)

	router.GET("/request/status", h.GetRequestStatus)
	router.POST("/request/status", h.ChangeRequestStatus)
}

func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./resources/static")
	router.Static("/img", "./resources/img")
}

func (h *Handler) errorHandler(ctx *gin.Context, statusCode int, err error) {
	logrus.Error(err.Error())
	ctx.JSON(statusCode, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}

func (h *Handler) getUserID(c *gin.Context) uint {
	userIDStr, err := c.Cookie("user_id")
	if err != nil {
		return 1
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 1
	}
	return uint(userID)
}

func (h *Handler) isLoggedIn(c *gin.Context) bool {
	_, err := c.Cookie("user_id")
	return err == nil
}

func (h *Handler) getUserLogin(c *gin.Context) string {
	login, err := c.Cookie("user_login")
	if err != nil {
		return ""
	}
	return login
}

func (h *Handler) GetServiceList(c *gin.Context) {
	services, err := h.repo.GetAllRadiationRanges()
	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	userID := h.getUserID(c)
	count := h.repo.GetRequestItemCount(userID)
	loggedIn := h.isLoggedIn(c)
	login := h.getUserLogin(c)

	c.HTML(http.StatusOK, "services.html", gin.H{
		"services":  services,
		"count":     count,
		"user_id":   userID,
		"logged_in": loggedIn,
		"login":     login,
	})
}

func (h *Handler) GetServiceDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	service, err := h.repo.GetRadiationRangeByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	videoURL := service.ImageURL
	videoURL = strings.Replace(videoURL, ".jpg", ".mp4", 1)
	videoURL = strings.Replace(videoURL, ".png", ".mp4", 1)

	userID := h.getUserID(c)
	count := h.repo.GetRequestItemCount(userID)
	loggedIn := h.isLoggedIn(c)
	login := h.getUserLogin(c)

	c.HTML(http.StatusOK, "service_detail.html", gin.H{
		"service":   service,
		"videoURL":  videoURL,
		"count":     count,
		"user_id":   userID,
		"logged_in": loggedIn,
		"login":     login,
	})
}

func (h *Handler) AddToRequest(c *gin.Context) {
	var input InputAddToRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	if input.ServiceID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID услуги"})
		return
	}

	userID := h.getUserID(c)
	req, err := h.repo.GetRequestByUserID(userID)
	if err != nil {
		req = &ds.ExperimentRequest{
			UserID: userID,
			Status: "draft",
		}
		if err := h.repo.CreateRequest(req); err != nil {
			h.errorHandler(c, http.StatusInternalServerError, err)
			return
		}
	}

	var existingItem ds.RequestItem
	if err := h.repo.GetDB().Where("request_id = ? AND radiation_id = ?", req.ID, input.ServiceID).First(&existingItem).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Услуга уже в заявке"})
		return
	}

	item := &ds.RequestItem{
		RequestID:    req.ID,
		RadiationID:  uint(input.ServiceID),
		Area:         input.Area,
		WorkFunction: input.WorkFunction,
		Efficiency:   input.Efficiency,
		Intensity:    input.Intensity,
		Comment:      input.Comment,
		Order:        input.Order,
	}

	if item.Area == 0 {
		item.Area = 10
	}
	if item.WorkFunction == 0 {
		item.WorkFunction = 2.3
	}
	if item.Efficiency == 0 {
		item.Efficiency = 18
	}
	if item.Intensity == 0 {
		item.Intensity = 100
	}

	item.CalculatedCurrent = item.Intensity * item.Area * item.Efficiency / 1000

	if err := h.repo.AddItemToRequest(item); err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	count := h.repo.GetRequestItemCount(userID)
	c.JSON(http.StatusOK, gin.H{
		"status":     "added",
		"count":      count,
		"request_id": req.ID,
	})
}

func (h *Handler) GetRequest(c *gin.Context) {
	userID := h.getUserID(c)
	status := c.Query("status")
	if status == "" {
		status = "draft"
	}

	requestIDStr := c.Query("request_id")
	var requestID uint = 0
	if requestIDStr != "" && requestIDStr != "undefined" {
		id, err := strconv.Atoi(requestIDStr)
		if err == nil {
			requestID = uint(id)
		}
	}

	var items []ds.RequestItem
	var err error
	var currentStatus string = status

	if requestID > 0 {
		request, err := h.repo.GetRequestWithItems(requestID)
		if err == nil {
			items = request.Items
			currentStatus = request.Status
		}
	} else {
		items, err = h.repo.GetRequestItemsWithRadiationByUser(userID, status)
	}

	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	allRequests, _ := h.repo.GetUserAllRequests(userID)
	draftRequestID, _ := h.repo.GetDraftRequestID(userID)

	if c.GetHeader("Accept") == "application/json" || c.Query("format") == "json" {
		c.JSON(http.StatusOK, gin.H{
			"items":        items,
			"status":       currentStatus,
			"request_id":   draftRequestID,
			"all_requests": allRequests,
		})
		return
	}

	c.HTML(http.StatusOK, "request.html", gin.H{
		"items":        items,
		"user_id":      userID,
		"logged_in":    h.isLoggedIn(c),
		"login":        h.getUserLogin(c),
		"status":       currentStatus,
		"request_id":   draftRequestID,
		"all_requests": allRequests,
	})
}

func (h *Handler) UpdateRequest(c *gin.Context) {
	var input InputUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	radiationID := input.RadiationID
	if radiationID == 0 {
		radiationID = input.ServiceID
	}

	if radiationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID диапазона"})
		return
	}

	userID := h.getUserID(c)
	req, err := h.repo.GetRequestByUserID(userID)
	if err != nil {
		req = &ds.ExperimentRequest{
			UserID: userID,
			Status: "draft",
		}
		if err := h.repo.CreateRequest(req); err != nil {
			h.errorHandler(c, http.StatusInternalServerError, err)
			return
		}
	}

	item := &ds.RequestItem{
		RequestID:    req.ID,
		RadiationID:  radiationID,
		Area:         input.Area,
		WorkFunction: input.WorkFunction,
		Efficiency:   input.Efficiency,
		Intensity:    input.Intensity,
		Comment:      input.Comment,
		Order:        input.Order,
	}

	if item.Intensity == 0 {
		item.Intensity = 100
	}

	item.CalculatedCurrent = item.Intensity * item.Area * item.Efficiency / 1000

	if err := h.repo.UpdateItemInRequest(item); err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "updated",
		"item":   item,
	})
}

func (h *Handler) ChangeRequestStatus(c *gin.Context) {
	var input struct {
		RequestID uint   `json:"request_id"`
		Status    string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}
	validStatuses := map[string]bool{
		"draft":       true,
		"удалён":      true,
		"сформирован": true,
		"завершён":    true,
		"отклонён":    true,
	}
	if !validStatuses[input.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый статус"})
		return
	}
	if input.Status == "сформирован" {
		h.repo.CalculateAndSaveTotalCurrent(input.RequestID)
		// ✅ Очищаем корзину - создаём новую заявку в статусе черновик
		userID := h.getUserID(c)
		newReq := &ds.ExperimentRequest{
			UserID: userID,
			Status: "draft",
		}
		h.repo.CreateRequest(newReq)
	}
	if err := h.repo.UpdateRequestStatus(input.RequestID, input.Status); err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":         "changed",
		"new_status":     input.Status,
		"new_request_id": input.RequestID,
	})
}
func (h *Handler) GetRequestStatus(c *gin.Context) {
	userID := h.getUserID(c)
	status, err := h.repo.GetUserRequestStatus(userID)
	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	requestID, _ := h.repo.GetDraftRequestID(userID)
	allRequests, _ := h.repo.GetUserAllRequests(userID)

	c.JSON(http.StatusOK, gin.H{
		"status":       status,
		"request_id":   requestID,
		"user_id":      userID,
		"all_requests": allRequests,
	})
}

func (h *Handler) RemoveFromRequest(c *gin.Context) {
	var reqBody struct {
		ServiceID   int `json:"service_id"`
		RadiationID int `json:"radiation_id"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	userID := h.getUserID(c)
	radiationID := uint(reqBody.RadiationID)
	if radiationID == 0 {
		radiationID = uint(reqBody.ServiceID)
	}

	req, err := h.repo.GetRequestByUserID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	h.repo.RemoveItemFromRequest(req.ID, radiationID)

	count := h.repo.GetRequestItemCount(userID)
	c.JSON(http.StatusOK, gin.H{
		"status": "removed",
		"count":  count,
	})
}

func (h *Handler) DeleteRequest(c *gin.Context) {
	var reqBody struct {
		RequestID uint `json:"request_id"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	err := h.repo.DeleteRequestSQL(reqBody.RequestID)
	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
