package repository

import (
	"RIP_Golab/internal/app/ds"
	"fmt"
)

// GetCalculationByPhysicistID - получить заявку пользователя (только черновик)
func (r *Repository) GetCalculationByPhysicistID(physicistID uint) (*ds.RadiationCalculation, error) {
	var calc ds.RadiationCalculation
	err := r.db.Where("physicist_id = ? AND status = ?", physicistID, "draft").
		First(&calc).Error
	return &calc, err
}

// GetCalculationByID - получить заявку по ID (с подгрузкой услуг)
func (r *Repository) GetCalculationByID(id uint) (*ds.RadiationCalculation, error) {
	var calc ds.RadiationCalculation
	err := r.db.Preload("Items.Radiation").
		Where("id = ? AND status != ?", id, "удалён").
		First(&calc).Error
	return &calc, err
}

// CreateCalculation - создание новой заявки
func (r *Repository) CreateCalculation(calc *ds.RadiationCalculation) error {
	return r.db.Create(calc).Error
}

// AddItemToCalculation - добавление позиции в заявку (ORM)
func (r *Repository) AddItemToCalculation(item *ds.CalculationItem) error {
	return r.db.Create(item).Error
}

// GetCalculationItemCount - получить количество позиций в заявке
func (r *Repository) GetCalculationItemCount(physicistID uint) int64 {
	var count int64
	r.db.Model(&ds.CalculationItem{}).
		Joins("JOIN radiation_calculations ON radiation_calculations.id = calculation_items.calculation_id").
		Where("radiation_calculations.physicist_id = ? AND radiation_calculations.status = ?", physicistID, "draft").
		Count(&count)
	return count
}

// DeleteCalculationSQL - логическое удаление заявки через SQL UPDATE
func (r *Repository) DeleteCalculationSQL(calcID uint) error {
	result := r.db.Exec("UPDATE radiation_calculations SET status = 'удалён' WHERE id = ?", calcID)
	if result.RowsAffected == 0 {
		return fmt.Errorf("заявка не найдена")
	}
	return result.Error
}

// GetAllRadiationRanges - получение всех услуг
func (r *Repository) GetAllRadiationRanges() ([]ds.RadiationRange, error) {
	var ranges []ds.RadiationRange
	err := r.db.Where("is_delete = false").Find(&ranges).Error
	return ranges, err
}

// SearchRadiationRangesByName - поиск по названию
func (r *Repository) SearchRadiationRangesByName(name string) ([]ds.RadiationRange, error) {
	var ranges []ds.RadiationRange
	err := r.db.Where("name ILIKE ? AND is_delete = false", "%"+name+"%").Find(&ranges).Error
	return ranges, err
}

// GetRadiationRangeByID - получение конкретной услуги
func (r *Repository) GetRadiationRangeByID(id uint) (*ds.RadiationRange, error) {
	var rad ds.RadiationRange
	err := r.db.Where("id = ? AND is_delete = false", id).First(&rad).Error
	return &rad, err
}
