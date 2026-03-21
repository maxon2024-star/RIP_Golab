package ds

type CalculationItem struct {
	ID                uint    `gorm:"primaryKey" json:"id"`
	CalculationID     uint    `gorm:"not null" json:"calculation_id"`
	RadiationID       uint    `gorm:"not null" json:"radiation_id"`
	Area              float64 `gorm:"type:decimal(10,2)" json:"area"`               // Площадь (вводится юзером)
	Efficiency        float64 `gorm:"type:decimal(5,2)" json:"efficiency"`          // КПД (вводится юзером)
	Frequency         float64 `gorm:"type:decimal(20,2)" json:"frequency"`          // Частота (вводится юзером)
	WorkFunction      float64 `gorm:"type:decimal(10,2)" json:"work_function"`      // Энергия выхода (вводится юзером)
	KineticEnergy     float64 `gorm:"type:decimal(10,2)" json:"kinetic_energy"`     // Рассчитывается
	CalculatedCurrent float64 `gorm:"type:decimal(15,5)" json:"calculated_current"` // Рассчитывается

	Radiation RadiationRange `gorm:"foreignKey:RadiationID" json:"radiation"`
}

func (CalculationItem) TableName() string {
	return "calculation_items"
}
