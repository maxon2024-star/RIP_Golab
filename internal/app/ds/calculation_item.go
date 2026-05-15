package ds

type CalculationItem struct {
	ID                uint    `gorm:"primaryKey" json:"id"`
	CalculationID     uint    `gorm:"not null" json:"calculation_id"`
	RadiationID       uint    `gorm:"not null" json:"radiation_id"`
	Frequency         float64 `gorm:"type:decimal(20,4)" json:"frequency"`
	WorkFunction      float64 `gorm:"type:decimal(10,4)" json:"work_function"`
	Area              float64 `gorm:"type:decimal(10,4)" json:"area"`
	Efficiency        float64 `gorm:"type:decimal(5,2)" json:"efficiency"`
	CalculatedCurrent float64 `gorm:"type:decimal(15,4)" json:"calculated_current"`
	IsPriority        bool    `gorm:"default:false" json:"is_priority"`

	Radiation   RadiationRange       `gorm:"foreignKey:RadiationID" json:"radiation"`
	Calculation RadiationCalculation `gorm:"foreignKey:CalculationID" json:"calculation"`
}

func (CalculationItem) TableName() string {
	return "calculation_items"
}
