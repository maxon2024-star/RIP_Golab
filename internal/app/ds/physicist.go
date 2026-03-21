package ds

import "RIP_Golab/internal/app/role"

type Physicist struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	Login    string    `gorm:"type:varchar(25);unique;not null" json:"login"`
	Password string    `gorm:"type:varchar(100);not null" json:"-"`
	Role     role.Role `gorm:"type:int;default:1" json:"role"` // 1 - Физик, 2 - Профессор
}

func (Physicist) TableName() string {
	return "physicists"
}
