package ds

import "time"

type RadiationCalculation struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	ModeratorID *uint      `gorm:"default:null;index" json:"moderator_id"`
	Status      string     `gorm:"type:varchar(20);default:'draft';index" json:"status"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	FormedAt    *time.Time `gorm:"default:null" json:"formed_at"`
	CompletedAt *time.Time `gorm:"default:null" json:"completed_at"`

	Description  string  `gorm:"type:text" json:"description"`
	TotalCurrent float64 `gorm:"type:decimal(10,4);default:0" json:"total_current"`

	User  User              `gorm:"foreignKey:UserID" json:"user"`
	Items []CalculationItem `gorm:"foreignKey:CalculationID;constraint:OnDelete:CASCADE;" json:"items"`
}

func (RadiationCalculation) TableName() string {
	return "radiation_calculations"
}
