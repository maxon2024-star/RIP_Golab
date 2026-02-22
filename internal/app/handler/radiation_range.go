package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRadiationRanges - получение списка всех диапазонов
func (h *Handler) GetRadiationRanges(ctx *gin.Context) {
	ranges, err := h.repo.GetAllRadiationRanges()
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	// ✅ Исправлено: GetRequestCount -> GetRequestItemCount с userID
	count := h.repo.GetRequestItemCount(1) // Хардкод userID = 1 (потом будет из JWT)
	ctx.HTML(http.StatusOK, "services.html", gin.H{
		"services": ranges,
		"count":    count,
	})
}

// GetRadiationRangeDetail - получение деталей диапазона
func (h *Handler) GetRadiationRangeDetail(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	radRange, err := h.repo.GetRadiationRangeByID(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}
	// Конвертация изображения в видео URL
	videoURL := radRange.VideoURL
	if videoURL == "" {
		videoURL = radRange.ImageURL
		videoURL = strings.Replace(videoURL, ".jpg", ".mp4", 1)
		videoURL = strings.Replace(videoURL, ".png", ".mp4", 1)
	}
	// ✅ Исправлено: GetRequestCount -> GetRequestItemCount с userID
	count := h.repo.GetRequestItemCount(1) // Хардкод userID = 1
	ctx.HTML(http.StatusOK, "service_detail.html", gin.H{
		"service":  radRange,
		"videoURL": videoURL,
		"count":    count,
	})
}
