package ds

type RadiationRange struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	Name         string  `gorm:"type:varchar(50);not null" json:"name"`
	Description  string  `gorm:"type:text" json:"description"`
	ImageURL     string  `gorm:"type:varchar(255)" json:"image_url"`
	VideoURL     string  `gorm:"type:varchar(255)" json:"video_url"`
	Wavelength   string  `gorm:"type:varchar(50)" json:"wavelength"`
	EnergyRange  string  `gorm:"type:varchar(50)" json:"energy_range"`
	Frequency    string  `gorm:"type:varchar(50)" json:"frequency"`
	Intensity    float64 `gorm:"type:decimal(10,2);default:100" json:"intensity"`
	WorkFunction float64 `gorm:"type:decimal(10,4);default:2.0" json:"work_function"` // Перенесли сюда
	IsDelete     bool    `gorm:"type:boolean;default:false" json:"is_delete"`
}

func (RadiationRange) TableName() string {
	return "radiation_ranges"
}
