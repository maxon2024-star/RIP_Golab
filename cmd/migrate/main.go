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
		&ds.Physicist{},
		&ds.RadiationRange{},
		&ds.RadiationCalculation{},
		&ds.CalculationItem{},
	)
	if err != nil {
		log.Fatal("❌ Error migrating database:", err)
	}
	log.Println("✅ Миграции выполнены!")

	SeedTestData(db)
}

func SeedTestData(db *gorm.DB) {
	var userCount int64
	db.Model(&ds.Physicist{}).Count(&userCount)
	if userCount == 0 {
		users := []ds.Physicist{
			{Login: "user", Password: "user", Role: 1},
			{Login: "admin", Password: "admin", Role: 2},
		}
		for _, u := range users {
			db.Create(&u)
		}
	}

	var count int64
	db.Model(&ds.RadiationRange{}).Count(&count)
	if count == 0 {
		// ShortDescription содержит английский текст для нейросети CLIP
		// Description содержит длинное русское описание для вывода в карточках
		ranges := []ds.RadiationRange{
			{
				Name:             "Радиоволны",
				ShortDescription: "Electromagnetic waves with the longest wavelengths, used for long-distance radio communication.",
				Description:      "Диапазон ЭМИ, применяемый для связи и радаров. Имеет самую большую длину волны.",
				ImageURL:         "http://localhost:9000/physicsservice/radio.jpg", VideoURL: "http://localhost:9000/physicsservice/radio.mp4",
				IsDelete: false,
			},
			{
				Name:             "Инфракрасное излучение",
				ShortDescription: "Invisible radiant energy, electromagnetic radiation with longer wavelengths than visible light.",
				Description:      "Тепловое излучение объектов, невидимое для глаза, но ощущаемое как тепло.",
				ImageURL:         "http://localhost:9000/physicsservice/infrared.jpg", VideoURL: "http://localhost:9000/physicsservice/infrared.mp4",
				IsDelete: false,
			},
			{
				Name:             "Видимый свет",
				ShortDescription: "The portion of the electromagnetic spectrum that is visible to the human eye, enabling human sight.",
				Description:      "Базовое видимое отраженное излучение объектов, воспринимаемое человеческим глазом.",
				ImageURL:         "http://localhost:9000/physicsservice/visible.jpg", VideoURL: "http://localhost:9000/physicsservice/visible.mp4",
				IsDelete: false,
			},
			{
				Name:             "Ультрафиолет",
				ShortDescription: "Invisible electromagnetic radiation emitted by the sun, responsible for summer tans and sunburns.",
				Description:      "Невидимая часть солнечного спектра, вызывающая флуоресценцию и используемая в медицине.",
				ImageURL:         "http://localhost:9000/physicsservice/uv.jpg", VideoURL: "http://localhost:9000/physicsservice/uv.mp4",
				IsDelete: false,
			},
			{
				Name:             "Рентген",
				ShortDescription: "High-energy electromagnetic radiation capable of passing through many materials, used in medicine.",
				Description:      "Самые высокоэнергетичные частицы для глубокого сканирования материалов и медицинских исследований.",
				ImageURL:         "http://localhost:9000/physicsservice/xray.jpg", VideoURL: "http://localhost:9000/physicsservice/xray.mp4",
				IsDelete: false,
			},
		}
		for _, r := range ranges {
			db.Create(&r)
		}
		log.Println("✅ Диапазоны добавлены!")
	}
}
