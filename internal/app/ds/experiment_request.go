package ds

import "time"

// ExperimentRequest - заявка на расчет фотоэффекта
type ExperimentRequest struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"not null;index" json:"user_id"`
	Status       string     `gorm:"type:varchar(20);default:'draft';index" json:"status"`
	TotalCurrent float64    `gorm:"type:decimal(10,4);default:0" json:"total_current"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	FinishedAt   *time.Time `gorm:"default:null" json:"finished_at"`
	ModeratorID  *uint      `gorm:"default:null;index" json:"moderator_id"`

	User  User          `gorm:"foreignKey:UserID" json:"user"`
	Items []RequestItem `gorm:"foreignKey:RequestID" json:"items"`
}

// TableName - имя таблицы в БД
func (ExperimentRequest) TableName() string {
	return "experiment_requests"
}
