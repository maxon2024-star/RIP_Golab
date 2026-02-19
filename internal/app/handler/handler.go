package handler

import (
	"RIP_Golab/internal/app/repository"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	repo *repository.Repository
}

func NewHandler() *Handler {
	return &Handler{
		repo: repository.GetInstance(),
	}
}

// ExtendedRequestItem - для отображения в заявке
type ExtendedRequestItem struct {
	repository.RequestItem
	Name        string  `json:"name"`
	ImageURL    string  `json:"image_url"`
	EnergyRange string  `json:"energy_range"`
	Wavelength  string  `json:"wavelength"`
	Frequency   string  `json:"frequency"`
	Intensity   float64 `json:"intensity"`
}

func (h *Handler) GetServiceList(c *gin.Context) {
	services := h.repo.GetServices()
	count := h.repo.GetRequestCount()
	c.HTML(http.StatusOK, "services.html", gin.H{
		"services": services,
		"count":    count,
	})
}

func (h *Handler) GetServiceDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	service, err := h.repo.GetServiceByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	videoURL := service.ImageURL
	videoURL = strings.Replace(videoURL, ".jpg", ".mp4", 1)
	videoURL = strings.Replace(videoURL, ".png", ".mp4", 1)
	videoURL = strings.Replace(videoURL, ".jpeg", ".mp4", 1)

	count := h.repo.GetRequestCount()

	c.HTML(http.StatusOK, "service_detail.html", gin.H{
		"service":  service,
		"videoURL": videoURL,
		"count":    count,
	})
}

func (h *Handler) GetRequest(c *gin.Context) {
	requestItems := h.repo.GetRequest()
	services := h.repo.GetServices()

	// Создаем мапу для быстрого поиска сервисов
	servicesMap := make(map[int]repository.Service)
	for _, s := range services {
		servicesMap[s.ID] = s
	}

	// Объединяем данные запроса с данными сервисов
	extendedItems := make([]ExtendedRequestItem, 0, len(requestItems))
	for _, item := range requestItems {
		service, exists := servicesMap[item.ServiceID]
		if exists {
			extendedItem := ExtendedRequestItem{
				RequestItem: item,
				Name:        service.Name,
				ImageURL:    service.ImageURL,
				EnergyRange: service.EnergyRange,
				Wavelength:  service.Wavelength,
				Frequency:   service.Frequency,
				Intensity:   service.Intensity,
			}
			extendedItems = append(extendedItems, extendedItem)
		}
	}

	// Проверяем, нужен ли JSON (для AJAX запросов)
	if c.GetHeader("Accept") == "application/json" || c.Query("format") == "json" {
		c.JSON(http.StatusOK, gin.H{
			"items": extendedItems,
		})
		return
	}

	// Для HTML страницы
	c.HTML(http.StatusOK, "request.html", gin.H{
		"items": extendedItems,
	})
}

func (h *Handler) AddToRequest(c *gin.Context) {
	var item repository.RequestItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.repo.AddToRequest(item)

	c.JSON(http.StatusOK, gin.H{
		"status": "added",
		"count":  h.repo.GetRequestCount(),
	})
}

func (h *Handler) UpdateRequest(c *gin.Context) {
	var item repository.RequestItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.repo.UpdateRequestItem(item)

	c.JSON(http.StatusOK, gin.H{
		"status": "updated",
	})
}

func (h *Handler) RemoveFromRequest(c *gin.Context) {
	var reqBody struct {
		ServiceID int `json:"service_id"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.repo.RemoveFromRequest(reqBody.ServiceID)

	c.JSON(http.StatusOK, gin.H{
		"status": "removed",
		"count":  h.repo.GetRequestCount(),
	})
}
