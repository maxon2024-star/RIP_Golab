package ds

type CalculationItem struct {
	ID            uint `gorm:"primaryKey" json:"id"`
	CalculationID uint `gorm:"not null;index;uniqueIndex:idx_calc_rad" json:"calculation_id"`
	RadiationID   uint `gorm:"not null;index;uniqueIndex:idx_calc_rad" json:"radiation_id"`

	Area              float64 `gorm:"type:decimal(10,2);default:10" json:"area"`
	Efficiency        float64 `gorm:"type:decimal(5,2);default:18" json:"efficiency"`
	CalculatedCurrent float64 `gorm:"type:decimal(10,4)" json:"calculated_current"`

	// ТЕПЕРЬ ЭТО РЕАЛЬНЫЕ КОЛОНКИ В БД! (убрали gorm:"-")
	Frequency    float64 `gorm:"type:decimal(30,4)" json:"frequency"`
	WorkFunction float64 `gorm:"type:decimal(10,4)" json:"work_function"`

	// Виртуальное поле (оставляем, так как это просто результат вычислений)
	KineticEnergy float64 `gorm:"-" json:"-"`

	Calculation RadiationCalculation `gorm:"foreignKey:CalculationID" json:"-"`
	Radiation   RadiationRange       `gorm:"foreignKey:RadiationID" json:"radiation"`
}

func (CalculationItem) TableName() string {
	return "calculation_items"
}
