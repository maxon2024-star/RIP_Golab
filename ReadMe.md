# Лабораторная работа №3 — Веб-сервис для SPA

[Методические указания](https://github.com/iu5git/Web/blob/main/tutorials/lab3-go/README.md) | [Репозиторий](https://github.com/maxon2024-star/RIP_Golab)

## Цель работы

Создание веб-сервиса на Golang с использованием фреймворка Gin для предоставления REST API клиентским приложениям (SPA). Реализация полной бизнес-логики предметной области "Расчёт фотоэффекта", подключение к PostgreSQL через ORM GORM, хранение файлов в MinIO.

## Технический стек

- **Язык:** Go 1.25.7
- **Фреймворк:** Gin v1.11.0
- **База данных:** PostgreSQL 15 (через Docker)
- **ORM:** GORM v1.31.1
- **Хранилище файлов:** MinIO v7
- **Конфигурация:** Viper + .env
- **Тестирование API:** Postman / Insomnia

## Структура проекта

```
RIP_Golab1/
├── cmd/
│   ├── migrate/          # Миграции БД
│   └── server/           # Точка входа сервера
├── internal/
│   ├── app/
│   │   ├── config/       # Конфигурация
│   │   ├── ds/           # Модели данных
│   │   ├── dsn/          # DSN строки
│   │   ├── handler/      # HTTP обработчики
│   │   └── repository/   # Репозиторий (ORM + MinIO)
│   └── pkg/
│       └── app.go        # Инициализация приложения
├── config/
│   └── config.toml       # TOML конфигурация
├── resources/
│   └── static/           # Статические файлы
├── templates/            # HTML шаблоны
├── .env                  # Переменные окружения
├── go.mod
└── README.md
```

## Модели данных

### Таблица users
| Поле | Тип | Описание |
|------|-----|----------|
| id | uint | Первичный ключ |
| login | varchar(25) | Уникальный логин |
| password | varchar(100) | Хэш пароля |
| is_moderator | boolean | Флаг модератора |

### Таблица radiation_ranges (услуги)
| Поле | Тип | Описание |
|------|-----|----------|
| id | uint | Первичный ключ |
| name | varchar(50) | Название диапазона |
| description | text | Описание |
| image_url | varchar(255) | Ссылка на изображение в MinIO |
| video_url | varchar(255) | Ссылка на видео в MinIO |
| wavelength | varchar(50) | Длина волны |
| energy_range | varchar(50) | Энергетический диапазон |
| frequency | varchar(50) | Частота |
| is_delete | boolean | Флаг логического удаления |

### Таблица radiation_calculations (заявки)
| Поле | Тип | Описание |
|------|-----|----------|
| id | uint | Первичный ключ |
| user_id | uint | Внешний ключ на users |
| moderator_id | uint | Внешний ключ на users (nullable) |
| status | varchar(20) | Статус заявки |
| created_at | timestamp | Дата создания |
| formed_at | timestamp | Дата формирования (nullable) |
| completed_at | timestamp | Дата завершения (nullable) |
| description | text | Описание заявки |
| total_current | decimal | Рассчитанный ток |

### Таблица calculation_items (м-м связь)
| Поле | Тип | Описание |
|------|-----|----------|
| id | uint | Первичный ключ |
| calculation_id | uint | Внешний ключ на radiation_calculations |
| radiation_id | uint | Внешний ключ на radiation_ranges |
| frequency | double | Частота излучения |
| work_function | decimal | Работа выхода |
| area | decimal | Площадь катода |
| efficiency | decimal | Квантовая эффективность |
| kinetic_energy | decimal | Кинетическая энергия (расчётное) |
| calculated_current | decimal | Ток фотоэффекта (расчётное) |

**Уникальный составной ключ:** (calculation_id, radiation_id)

## Статусы заявок

1. **draft** — черновик (создаёт пользователь)
2. **сформирован** — готов к модерации (создатель формирует)
3. **завершён** — одобрено (модератор завершает)
4. **отклонён** — отклонено (модератор отклоняет)
5. **удалён** — логически удалено (создатель удаляет)

**Правила перехода статусов:**
- Создатель: draft → сформирован, draft → удалён
- Модератор: сформирован → завершён, сформирован → отклонён

## REST API эндпоинты

Все методы имеют префикс `/api`. Авторизация через заголовок `Authorization: Bearer <token>`.

### Домен услуги

| Метод | URL | Описание |
|-------|-----|----------|
| GET | `/api/services` | Список услуг с фильтрацией по search |
| GET | `/api/services/:id` | Одна услуга по ID |
| POST | `/api/services` | Добавление услуги с изображением и видео (multipart/form-data) |

### Домен корзины (м-м связь)

| Метод | URL | Описание |
|-------|-----|----------|
| GET | `/api/cart/icon` | Иконка корзины (id черновика + количество услуг) |
| POST | `/api/cart` | Добавление услуги в заявку-черновик |
| PUT | `/api/cart` | Изменение параметров услуги в заявке (без PK м-м) |
| DELETE | `/api/cart?radiation_id=:id` | Удаление услуги из заявки (без PK м-м) |

### Домен заявки

| Метод | URL | Описание |
|-------|-----|----------|
| GET | `/api/requests` | Список заявок с фильтрацией по status, date_from, date_to |
| GET | `/api/requests/:id` | Одна заявка с услугами и изображениями |
| PUT | `/api/requests/:id` | Изменение полей заявки |
| PUT | `/api/requests/:id/form` | Сформировать заявку (вычисление формулы) |
| PUT | `/api/requests/:id/complete` | Завершить/отклонить заявку модератором |
| DELETE | `/api/requests/:id` | Логическое удаление заявки |

### Домен пользователь

| Метод | URL | Описание |
|-------|-----|----------|
| POST | `/api/register` | Регистрация нового пользователя |
| POST | `/api/login` | Аутентификация (заглушка для ЛР4) |
| POST | `/api/logout` | Деавторизация (заглушка для ЛР4) |

## Singleton функция пользователя

В текущей лабораторной работе пользователь зафиксирован константой через функцию-singleton:

```go
func CurrentUser() uint {
    return 1 // Пользователь "user"
}

func CurrentModerator() uint {
    return 2 // Пользователь "admin" (модератор)
}
```

Эти функции используются во всех методах API для определения создателя и модератора. Системные поля (id, статусы, даты) вычисляются на бэкенде и не передаются с клиента.

## Бизнес-логика расчёта

При формировании заявки вычисляется ток фотоэффекта по формуле:

1. Энергия фотона: `E = h × ν` (h — постоянная Планка, ν — частота)
2. Работа выхода: `A = work_function × e` (e — заряд электрона)
3. Кинетическая энергия: `E_k = E - A`
4. Ток насыщения: `I = N_e × e` (с учётом КПД и площади катода)

**Обработка ошибок:**
- `calculated_current = -1` — энергии фотона недостаточно (красная граница)
- `calculated_current = -2` — некорректная частота (≤ 0)

## Конфигурация

### .env файл
```
DB_HOST=localhost
DB_PORT=5434
DB_NAME=photoeffect_db
DB_USER=root
DB_PASS=rootpassword123
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=root
MINIO_SECRET_KEY=rootpassword
MINIO_BUCKET_NAME=physicsservice
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8000
```

### config/config.toml
```toml
ServiceHost = "0.0.0.0"
ServicePort = 8000

[PhotoEffectConfig]
AppName = "Расчёт фотоэффекта"
DefaultArea = 10
DefaultWorkFunction = 2.3
DefaultEfficiency = 18
```

## Запуск проекта

### 1. Подготовка окружения
```bash
# Установить зависимости
go mod download

# Запустить Docker контейнеры (PostgreSQL, MinIO)
docker-compose up -d
```

### 2. Миграция базы данных
```bash
go run cmd/migrate/main.go
```

### 3. Запуск сервера
```bash
go run cmd/server/main.go
```

Сервер доступен по адресу: `http://localhost:8000`

## Тестирование в Postman/Insomnia

### Коллекция из 16 запросов для демонстрации

1. **GET** `/api/services?search=свет` — список услуг с фильтром
2. **GET** `/api/services/3` — одна услуга
3. **GET** `/api/cart/icon` — иконка корзины (пустая)
4. **GET** `/api/requests?status=сформирован&date_from=2025-01-01` — список заявок
5. **POST** `/api/services` — добавление услуги с файлами (multipart)
6. **POST** `/api/cart` — добавление услуги в черновик
7. **POST** `/api/cart` — добавление второй услуги в черновик
8. **GET** `/api/cart/icon` — иконка корзины (с услугами)
9. **GET** `/api/requests/1` — просмотр заявки с услугами
10. **PUT** `/api/cart` — изменение параметров услуги в заявке
11. **PUT** `/api/requests/1` — изменение полей заявки
12. **PUT** `/api/requests/1/complete` — завершение (ошибка, статус draft)
13. **PUT** `/api/requests/1/form` — сформировать заявку (расчёт формулы)
14. **PUT** `/api/requests/1/complete` — завершить сформированную заявку
15. **DELETE** `/api/requests/1` — логическое удаление заявки
16. **POST** `/api/register` — регистрация нового пользователя

### Примеры запросов

**Добавление услуги с файлами:**
```
POST /api/services
Content-Type: multipart/form-data

name: Гамма-излучение
description: Высокоэнергетическое излучение
image: [файл gamma.jpg]
video: [файл gamma.mp4]
```

**Добавление в корзину:**
```json
POST /api/cart
{
  "radiation_id": 3,
  "area": 15.5,
  "efficiency": 20.0,
  "work_function": 2.3,
  "frequency": 550000000000000
}
```

## Проверка данных в БД

```sql
-- Услуги
SELECT id, name, image_url, is_delete FROM radiation_ranges;

-- Заявки (включая удалённые)
SELECT id, user_id, status, formed_at, completed_at, total_current 
FROM radiation_calculations;

-- Связь м-м с расчётными полями
SELECT ci.id, ci.calculation_id, ci.radiation_id, ci.calculated_current, 
       rr.name as radiation_name
FROM calculation_items ci
JOIN radiation_ranges rr ON ci.radiation_id = rr.id;

-- Вычисляемое поле valid_results_count
SELECT rc.id, rc.status,
       COUNT(CASE WHEN ci.calculated_current > 0 THEN 1 END) as valid_results_count
FROM radiation_calculations rc
LEFT JOIN calculation_items ci ON rc.id = ci.calculation_id
GROUP BY rc.id, rc.status;
```

## Связи моделей

| Тип связи | Реализация |
|-----------|------------|
| Методы используют разные модели | Handler вызывает разные репозитории |
| Модели используют другие модели | GORM ForeignKey (CalculationItem.Radiation → RadiationRange) |
| Модели используют несколько таблиц | M-M связь через CalculationItem |

## Требования методички

- [x] REST API с префиксом `/api`
- [x] 16 запросов для демонстрации
- [x] Singleton функция пользователя (константа)
- [x] MinIO для изображений и видео услуг
- [x] ORM GORM для взаимодействия с БД
- [x] Статусы заявок по правилам (создатель/модератор)
- [x] Фильтрация на бэкенде (services по search, requests по date/status)
- [x] Вычисляемые поля (total_current, valid_results_count)
- [x] Нет POST заявки (создаётся автоматически при добавлении услуги)
- [x] Логическое удаление (статус "удалён" не передаётся клиенту)
- [x] Системные поля вычисляются на бэкенде

## Контрольные вопросы

1. **Веб-сервис** — программа, предоставляющая функциональность другим приложениям через сеть
2. **REST** — архитектурный стиль для построения распределённых систем на основе HTTP
3. **RPC** — протокол удалённого вызова процедур, альтернатива REST
4. **Заголовки HTTP** — метаданные запроса/ответа (Content-Type, Authorization, etc.)
5. **Методы HTTP** — GET, POST, PUT, DELETE, PATCH
6. **Версии HTTP** — 1.0, 1.1, 2, 3 (QUIC)
7. **HTTPS** — защищённая версия HTTP с шифрованием TLS/SSL
8. **Модель OSI** — 7-уровневая эталонная модель взаимодействия открытых систем

## Авторы

- Максим Задеба
- Лабораторная работа по курсу "Разработка интернет-приложений"