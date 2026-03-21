package ds

import "time"

type RadiationCalculation struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	PhysicistID uint       `gorm:"not null;index" json:"physicist_id"`
	ProfessorID *uint      `gorm:"default:null;index" json:"professor_id"`
	Status      string     `gorm:"type:varchar(20);default:'draft';index" json:"status"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	FormedAt    *time.Time `gorm:"default:null" json:"formed_at"`
	CompletedAt *time.Time `gorm:"default:null" json:"completed_at"`

	Description  string  `gorm:"type:text" json:"description"`
	TotalCurrent float64 `gorm:"type:decimal(10,4);default:0" json:"total_current"`

	Physicist Physicist         `gorm:"foreignKey:PhysicistID" json:"physicist"`
	Items     []CalculationItem `gorm:"foreignKey:CalculationID;constraint:OnDelete:CASCADE;" json:"items"`
}

func (RadiationCalculation) TableName() string {
	return "radiation_calculations"
}
