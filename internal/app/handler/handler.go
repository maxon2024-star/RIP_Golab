package handler

import (
	"RIP_Golab/internal/app/repository"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type Handler struct {
	repo *repository.Repository
}

func NewHandler() *Handler {
	return &Handler{
		repo: repository.GetInstance(),
	}
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

	c.HTML(http.StatusOK, "service_detail.html", gin.H{
		"service": service,
	})
}

func (h *Handler) GetRequest(c *gin.Context) {
	request := h.repo.GetRequest()

	c.HTML(http.StatusOK, "request.html", gin.H{
		"items": request,
	})
}

func (h *Handler) AddToRequest(c *gin.Context) {
	var item repository.RequestItem

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.repo.AddToRequest(item)
	c.JSON(http.StatusOK, gin.H{"status": "added", "count": h.repo.GetRequestCount()})
}

func (h *Handler) UpdateRequest(c *gin.Context) {
	var item repository.RequestItem

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.repo.UpdateRequestItem(item)
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
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
	c.JSON(http.StatusOK, gin.H{"status": "removed", "count": h.repo.GetRequestCount()})
}
