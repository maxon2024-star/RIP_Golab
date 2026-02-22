package repository

import (
	"RIP_Golab/internal/app/ds"
)

// GetRequestByUserID - получить заявку пользователя (только черновик)
func (r *Repository) GetRequestByUserID(userID uint) (*ds.ExperimentRequest, error) {
	var request ds.ExperimentRequest
	err := r.db.Where("user_id = ? AND status = ?", userID, "draft").
		First(&request).Error
	return &request, err
}

// CreateRequest - создание новой заявки
func (r *Repository) CreateRequest(request *ds.ExperimentRequest) error {
	return r.db.Create(request).Error
}

// UpdateRequest - обновление заявки
func (r *Repository) UpdateRequest(request *ds.ExperimentRequest) error {
	return r.db.Save(request).Error
}

// AddItemToRequest - добавление позиции в заявку
func (r *Repository) AddItemToRequest(item *ds.RequestItem) error {
	return r.db.Create(item).Error
}

// UpdateItemInRequest - обновление позиции в заявке
func (r *Repository) UpdateItemInRequest(item *ds.RequestItem) error {
	var existing ds.RequestItem
	err := r.db.Where("request_id = ? AND radiation_id = ?", item.RequestID, item.RadiationID).
		First(&existing).Error
	if err != nil {
		// ✅ Создаём новую запись если не найдена (без лишнего лога)
		return r.db.Create(item).Error
	}
	return r.db.Model(&ds.RequestItem{}).
		Where("request_id = ? AND radiation_id = ?", item.RequestID, item.RadiationID).
		Updates(map[string]interface{}{
			"area":               item.Area,
			"work_function":      item.WorkFunction,
			"efficiency":         item.Efficiency,
			"intensity":          item.Intensity,
			"comment":            item.Comment,
			"calculated_current": item.CalculatedCurrent,
			"order":              item.Order,
		}).Error
}

// RemoveItemFromRequest - удаление позиции из заявки
func (r *Repository) RemoveItemFromRequest(requestID, radiationID uint) error {
	return r.db.Where("request_id = ? AND radiation_id = ?", requestID, radiationID).
		Delete(&ds.RequestItem{}).Error
}

// GetRequestItemCount - получить количество позиций в заявке
func (r *Repository) GetRequestItemCount(userID uint) int64 {
	var count int64
	r.db.Model(&ds.RequestItem{}).
		Joins("JOIN experiment_requests ON experiment_requests.id = request_items.request_id").
		Where("experiment_requests.user_id = ? AND experiment_requests.status = ?", userID, "draft").
		Count(&count)
	return count
}

// GetRequestItemsWithRadiationByUser - получить позиции заявки с данными об излучении
// ИСПРАВЛЕНО: Используем Preload для загрузки связанной таблицы Radiation
func (r *Repository) GetRequestItemsWithRadiationByUser(userID uint, status string) ([]ds.RequestItem, error) {
	var request ds.ExperimentRequest
	// Сначала находим саму заявку, чтобы узнать её ID
	if err := r.db.Where("user_id = ? AND status = ?", userID, status).First(&request).Error; err != nil {
		return []ds.RequestItem{}, nil
	}

	var items []ds.RequestItem
	// Preload("Radiation") автоматически подтянет данные из таблицы radiation_ranges по foreign key
	err := r.db.Preload("Radiation").Where("request_id = ?", request.ID).Find(&items).Error
	if err != nil {
		return items, err
	}

	// ✅ Рассчитываем ток, если он не сохранён (логика осталась)
	for i := range items {
		if items[i].CalculatedCurrent == 0 {
			// Берем интенсивность либо из item, либо из загруженного Radiation
			intensity := items[i].Intensity
			if intensity == 0 && items[i].Radiation.ID != 0 {
				intensity = items[i].Radiation.Intensity
			}
			items[i].CalculatedCurrent = intensity * items[i].Area * items[i].Efficiency / 1000
		}
	}
	return items, err
}

// GetRequestWithItems - получить заявку с позициями для отображения
// ИСПРАВЛЕНО: Используем Preload("Items.Radiation")
func (r *Repository) GetRequestWithItems(requestID uint) (*ds.ExperimentRequest, error) {
	var request ds.ExperimentRequest
	// Одной командой грузим Заявку -> Позиции -> Излучение
	err := r.db.Preload("Items.Radiation").Where("id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}

	// Пересчет тока для безопасности
	for i := range request.Items {
		if request.Items[i].CalculatedCurrent == 0 {
			intensity := request.Items[i].Intensity
			if intensity == 0 && request.Items[i].Radiation.ID != 0 {
				intensity = request.Items[i].Radiation.Intensity
			}
			request.Items[i].CalculatedCurrent = intensity * request.Items[i].Area * request.Items[i].Efficiency / 1000
		}
	}
	return &request, err
}

// GetUserAllRequests - получить все заявки пользователя
func (r *Repository) GetUserAllRequests(userID uint) ([]ds.ExperimentRequest, error) {
	var requests []ds.ExperimentRequest
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// GetDraftRequestID - получить ID черновика заявки пользователя
func (r *Repository) GetDraftRequestID(userID uint) (uint, error) {
	var request ds.ExperimentRequest
	err := r.db.Where("user_id = ? AND status = ?", userID, "draft").First(&request).Error
	return request.ID, err
}

// GetUserRequestStatus - получить статус последней заявки пользователя
func (r *Repository) GetUserRequestStatus(userID uint) (string, error) {
	var request ds.ExperimentRequest
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&request).Error
	if err != nil {
		return "none", nil
	}
	return request.Status, nil
}

// UpdateRequestStatus - изменить статус заявки (через SQL UPDATE)
func (r *Repository) UpdateRequestStatus(requestID uint, status string) error {
	result := r.db.Exec("UPDATE experiment_requests SET status = ?, updated_at = NOW() WHERE id = ?", status, requestID)
	return result.Error
}

// CalculateAndSaveTotalCurrent - рассчитать и сохранить общий ток
func (r *Repository) CalculateAndSaveTotalCurrent(requestID uint) error {
	var total float64
	err := r.db.Model(&ds.RequestItem{}).
		Where("request_id = ?", requestID).
		Select("COALESCE(SUM(intensity * area * efficiency / 1000), 0)").
		Scan(&total).Error
	if err != nil {
		return err
	}
	result := r.db.Exec("UPDATE experiment_requests SET total_current = ?, updated_at = NOW() WHERE id = ?", total, requestID)
	return result.Error
}

// CalculateTotalCurrent - рассчитать общий ток фотоэффекта
func (r *Repository) CalculateTotalCurrent(requestID uint) (float64, error) {
	var total float64
	err := r.db.Model(&ds.RequestItem{}).
		Where("request_id = ?", requestID).
		Select("COALESCE(SUM(calculated_current), 0)").
		Scan(&total).Error
	return total, err
}
