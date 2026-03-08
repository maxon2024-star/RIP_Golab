package ds

type CalculationItem struct {
	ID            uint `gorm:"primaryKey" json:"id"`
	CalculationID uint `gorm:"not null;index;uniqueIndex:idx_calc_rad" json:"calculation_id"`
	RadiationID   uint `gorm:"not null;index;uniqueIndex:idx_calc_rad" json:"radiation_id"`

	Frequency    float64 `gorm:"type:double precision" json:"frequency"`
	WorkFunction float64 `gorm:"type:decimal(10,4);default:2.0" json:"work_function"`
	Area         float64 `gorm:"type:decimal(10,2);default:10" json:"area"`
	Efficiency   float64 `gorm:"type:decimal(5,2);default:18" json:"efficiency"`

	KineticEnergy     float64 `gorm:"type:decimal(10,4)" json:"kinetic_energy"` // E_k
	CalculatedCurrent float64 `gorm:"type:decimal(10,4)" json:"calculated_current"`

	Calculation RadiationCalculation `gorm:"foreignKey:CalculationID" json:"calculation"`
	Radiation   RadiationRange       `gorm:"foreignKey:RadiationID" json:"radiation"`
}

func (CalculationItem) TableName() string {
	return "calculation_items"
}
