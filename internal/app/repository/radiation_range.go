package repository

import "RIP_Golab/internal/app/ds"

func (r *Repository) GetAllRadiationRanges() ([]ds.RadiationRange, error) {
	var ranges []ds.RadiationRange
	err := r.db.Where("is_delete = false").Find(&ranges).Error
	return ranges, err
}

func (r *Repository) GetRadiationRangeByID(id uint) (*ds.RadiationRange, error) {
	var radRange ds.RadiationRange
	err := r.db.Where("is_delete = ?", false).First(&radRange, id).Error
	return &radRange, err
}

func (r *Repository) SearchRadiationRangesByName(name string) ([]ds.RadiationRange, error) {
	var ranges []ds.RadiationRange
	err := r.db.Where("name ILIKE ? AND is_delete = false", "%"+name+"%").Find(&ranges).Error
	return ranges, err
}
