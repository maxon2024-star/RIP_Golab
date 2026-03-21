package ds

type RadiationRange struct {
	ID          uint   `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	Name        string `gorm:"type:varchar(100);unique;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	ImageURL    string `gorm:"type:varchar(255)" json:"image_url"`
	VideoURL    string `gorm:"type:varchar(255)" json:"video_url"`
	IsDelete    bool   `gorm:"default:false" json:"is_delete"`
}

func (RadiationRange) TableName() string {
	return "radiation_ranges"
}
