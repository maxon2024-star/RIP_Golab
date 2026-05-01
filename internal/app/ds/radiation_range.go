package ds

type RadiationRange struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	Name             string `gorm:"type:varchar(100);not null" json:"name"`
	ShortDescription string `gorm:"type:varchar(255)" json:"short_description"` // Новое поле
	Description      string `gorm:"type:text" json:"description"`
	ImageURL         string `gorm:"type:varchar(255)" json:"image_url"`
	VideoURL         string `gorm:"type:varchar(255)" json:"video_url"`
	IsDelete         bool   `gorm:"default:false" json:"is_delete"`
}
