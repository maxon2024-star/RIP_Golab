package ds

type RequestItem struct {
	ID                uint    `gorm:"primaryKey" json:"id"`
	RequestID         uint    `gorm:"not null;index;uniqueIndex:idx_request_radiation" json:"request_id"`
	RadiationID       uint    `gorm:"not null;index;uniqueIndex:idx_request_radiation" json:"radiation_id"`
	Area              float64 `gorm:"type:decimal(10,2);default:10" json:"area"`
	WorkFunction      float64 `gorm:"type:decimal(5,2);default:2.3" json:"work_function"`
	Efficiency        float64 `gorm:"type:decimal(5,2);default:18" json:"efficiency"`
	Intensity         float64 `gorm:"type:decimal(10,2)" json:"intensity"`
	Comment           string  `gorm:"type:varchar(255)" json:"comment"`
	CalculatedCurrent float64 `gorm:"type:decimal(10,4)" json:"calculated_current"`
	Order             int     `gorm:"default:1" json:"order"`

	Request   ExperimentRequest `gorm:"foreignKey:RequestID" json:"request"`
	Radiation RadiationRange    `gorm:"foreignKey:RadiationID" json:"Radiation"`
}

func (RequestItem) TableName() string {
	return "request_items"
}
