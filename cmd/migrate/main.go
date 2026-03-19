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

	// Очищаем БД от старых таблиц принудительно (добавили calculation_items)
	db.Exec("DROP TABLE IF EXISTS request_items CASCADE")
	db.Exec("DROP TABLE IF EXISTS experiment_requests CASCADE")
	db.Exec("DROP TABLE IF EXISTS calculation_items CASCADE")
	db.Exec("DROP TABLE IF EXISTS radiation_calculations CASCADE")
	db.Exec("DROP TABLE IF EXISTS radiation_ranges CASCADE")
	db.Exec("DROP TABLE IF EXISTS users CASCADE")

	err = db.AutoMigrate(
		&ds.User{},
		&ds.RadiationRange{},
		&ds.RadiationCalculation{},
		&ds.CalculationItem{},
	)
	if err != nil {
		log.Fatal("❌ Error migrating database:", err)
	}
	log.Println("✅ Миграции выполнены!")

	// Строку с ALTER TABLE убрали, GORM сам создаст индекс из тега уникальности
	SeedTestData(db)
}

func SeedTestData(db *gorm.DB) {
	var userCount int64
	db.Model(&ds.User{}).Count(&userCount)
	if userCount == 0 {
		users := []ds.User{
			{Login: "user", Password: "user", IsModerator: false},
			{Login: "admin", Password: "admin", IsModerator: true},
		}
		for _, u := range users {
			db.Create(&u)
		}
	}

	var count int64
	db.Model(&ds.RadiationRange{}).Count(&count)
	if count == 0 {
		// ДОБАВИЛИ поле WorkFunction для каждой услуги
		ranges := []ds.RadiationRange{
			{
				Name: "Радиоволны", Description: "Диапазон ЭМИ",
				ImageURL: "http://localhost:9000/physicsservice/radio.jpg", VideoURL: "http://localhost:9000/physicsservice/radio.mp4",
				Wavelength: "1 мм - 100 км", EnergyRange: "12.4 мкэВ - 1.24 мэВ", Frequency: "3 кГц - 300 ГГц",
				WorkFunction: 2.0, IsDelete: false,
			},
			{
				Name: "Инфракрасное излучение", Description: "Тепловое излучение объектов",
				ImageURL: "http://localhost:9000/physicsservice/infrared.jpg", VideoURL: "http://localhost:9000/physicsservice/infrared.mp4",
				Wavelength: "700 нм - 1 мм", EnergyRange: "1.24 - 1.77 эВ", Frequency: "300 ГГц - 430 ТГц",
				WorkFunction: 2.1, IsDelete: false,
			},
			{
				Name: "Видимый свет", Description: "Базовое видимое отраженное излучение объектов",
				ImageURL: "http://localhost:9000/physicsservice/visible.jpg", VideoURL: "http://localhost:9000/physicsservice/visible.mp4",
				Wavelength: "380 - 700 нм", EnergyRange: "1.77 - 3.26 эВ", Frequency: "430 - 790 ТГц",
				WorkFunction: 2.2, IsDelete: false,
			},
			{
				Name: "Ультрафиолет", Description: "Невидимая часть солнечного спектра",
				ImageURL: "http://localhost:9000/physicsservice/uv.jpg", VideoURL: "http://localhost:9000/physicsservice/uv.mp4",
				Wavelength: "10 - 400 нм", EnergyRange: "3.1 - 124 эВ", Frequency: "790 ТГц - 30 ПГц",
				WorkFunction: 4.5, IsDelete: false,
			},
			{
				Name: "Рентген", Description: "Самые высокоэнергетичные частицы",
				ImageURL: "http://localhost:9000/physicsservice/xray.jpg", VideoURL: "http://localhost:9000/physicsservice/xray.mp4",
				Wavelength: "0.01 - 10 нм", EnergyRange: "124 эВ - 124 кэВ", Frequency: "30 ПГц - 30 ЭГц",
				WorkFunction: 5.0, IsDelete: false,
			},
		}
		for _, r := range ranges {
			db.Create(&r)
		}
		log.Println("✅ Диапазоны добавлены!")
	}
}
