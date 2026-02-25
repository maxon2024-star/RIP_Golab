package main

import (
	"RIP_Golab/internal/app/ds"
	"RIP_Golab/internal/app/dsn"
	"log"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	dsnString := dsn.FromEnv()
	db, err := gorm.Open(postgres.Open(dsnString), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Error connecting to database:", err)
	}

	// Миграция таблиц
	err = db.AutoMigrate(
		&ds.User{},
		&ds.RadiationRange{},
		&ds.ExperimentRequest{},
		&ds.RequestItem{},
	)
	if err != nil {
		log.Fatal("❌ Error migrating database:", err)
	}
	log.Println("✅ Миграции выполнены!")

	db.Exec("ALTER TABLE request_items ADD CONSTRAINT uk_request_radiation UNIQUE (request_id, radiation_id)")
	// Заполнение тестовыми данными
	SeedTestData(db)
}

func SeedTestData(db *gorm.DB) {
	// Проверяем, есть ли уже пользователи
	var userCount int64
	db.Model(&ds.User{}).Count(&userCount)
	if userCount == 0 {
		users := []ds.User{
			{ID: 1, Login: "user", Password: "user", IsModerator: false},
			{ID: 2, Login: "admin", Password: "admin", IsModerator: true},
		}
		for _, u := range users {
			db.Create(&u)
		}
		log.Println("✅ Пользователи добавлены!")
	}

	// Проверяем, есть ли уже данные о диапазонах
	var count int64
	db.Model(&ds.RadiationRange{}).Count(&count)
	if count == 0 {
		ranges := []ds.RadiationRange{
			{
				ID: 1, Name: "Радиоволны", Description: "Диапазон ЭМИ 1мм-100км",
				ImageURL:   "http://localhost:9000/physicsservice/radio.jpg",
				Wavelength: "1 мм - 100 км", EnergyRange: "12.4 мкэВ - 1.24 мэВ",
				Frequency: "3 кГц - 300 ГГц", Intensity: 10, IsDelete: false,
			},
			{
				ID: 2, Name: "Инфракрасное излучение", Description: "700нм-1мм",
				ImageURL:   "http://localhost:9000/physicsservice/infrared.jpg",
				Wavelength: "700 нм - 1 мм", EnergyRange: "1.24 - 1.77 эВ",
				Frequency: "300 ГГц - 430 ТГц", Intensity: 15, IsDelete: false,
			},
			{
				ID: 3, Name: "Видимый свет", Description: "380-700нм",
				ImageURL:   "http://localhost:9000/physicsservice/visible.jpg",
				Wavelength: "380 - 700 нм", EnergyRange: "1.77 - 3.26 эВ",
				Frequency: "430 - 790 ТГц", Intensity: 100, IsDelete: false,
			},
			{
				ID: 4, Name: "Ультрафиолет", Description: "10-400нм",
				ImageURL:   "http://localhost:9000/physicsservice/uv.jpg",
				Wavelength: "10 - 400 нм", EnergyRange: "3.1 - 124 эВ",
				Frequency: "790 ТГц - 30 ПГц", Intensity: 45, IsDelete: false,
			},
			{
				ID: 5, Name: "Рентген", Description: "0.01-10нм",
				ImageURL:   "http://localhost:9000/physicsservice/xray.jpg",
				Wavelength: "0.01 - 10 нм", EnergyRange: "124 эВ - 124 кэВ",
				Frequency: "30 ПГц - 30 ЭГц", Intensity: 5, IsDelete: false,
			},
		}
		for _, r := range ranges {
			db.Create(&r)
		}
		log.Println("✅ Диапазоны добавлены!")
	} else {
		log.Println("⚠️ Данные уже существуют, пропускаем сидирование.")
	}
}
