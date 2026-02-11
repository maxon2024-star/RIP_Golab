package repository

import (
	"errors"
	"sync"
)

type Service struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Wavelength  string  `json:"wavelength"`
	EnergyRange string  `json:"energy_range"`
	Frequency   string  `json:"frequency"`
	Intensity   float64 `json:"intensity"`
	ImageURL    string  `json:"image_url"`
}

type RequestItem struct {
	ServiceID    int     `json:"service_id"`
	Quantity     int     `json:"quantity"`
	Order        int     `json:"order"`
	Comment      string  `json:"comment"`
	Result       string  `json:"result"`
	Area         float64 `json:"area"`
	WorkFunction float64 `json:"work_function"`
	Efficiency   float64 `json:"efficiency"`
}

type Repository struct {
	services map[int]Service
	request  []RequestItem
	mutex    sync.RWMutex
}

var instance *Repository
var once sync.Once

func GetInstance() *Repository {
	once.Do(func() {
		// Внутри GetInstance():
		instance = &Repository{
			services: map[int]Service{
				1: {
					ID:          1,
					Name:        "Радиоизлучение",
					Description: "Радиоволны - это электромагнитные волны с самыми длинными волнами и самыми низкими частотами в электромагнитном спектре.",
					Wavelength:  "1 мм - 100 км",
					EnergyRange: "0.001 - 1.24 мэВ",
					Frequency:   "3 кГц - 300 МГц",
					Intensity:   0.0,
					ImageURL:    "http://localhost:9000/physicsservice/radio.jpg", // Измените на URL из MinIO
				},
				2: {
					ID:          2,
					Name:        "Инфракрасное излучение",
					Description: "Инфракрасное излучение лежит между видимым светом и радиоволнами в спектре электромагнитного излучения.",
					Wavelength:  "700 нм - 1 мм",
					EnergyRange: "1.24 - 1.77 эВ",
					Frequency:   "300 ГГц - 430 ТГц",
					Intensity:   15.0,
					ImageURL:    "http://localhost:9000/physicsservice/infrared.jpg", // Измените на URL из MinIO
				},
				3: {
					ID:          3,
					Name:        "Видимый свет",
					Description: "Видимый свет — это часть электромагнитного спектра, которую способно воспринимать человеческое глаз.",
					Wavelength:  "380 - 700 нм",
					EnergyRange: "1.77 - 3.26 эВ",
					Frequency:   "430 - 790 ТГц",
					Intensity:   100.0,
					ImageURL:    "http://localhost:9000/physicsservice/visible.jpg", // Измените на URL из MinIO
				},
				4: {
					ID:          4,
					Name:        "Ультрафиолетовое излучение",
					Description: "Ультрафиолетовое излучение имеет более короткие волны, чем видимый свет, и большую энергию фотонов.",
					Wavelength:  "10 - 400 нм",
					EnergyRange: "3.1 - 124 эВ",
					Frequency:   "790 ТГц - 30 ПГц",
					Intensity:   45.0,
					ImageURL:    "http://localhost:9000/physicsservice/uv.jpg", // Измените на URL из MinIO
				},
				5: {
					ID:          5,
					Name:        "Рентгеновское излучение",
					Description: "Рентгеновские лучи обладают высокой энергией и могут проникать сквозь многие материалы.",
					Wavelength:  "0.01 - 10 нм",
					EnergyRange: "124 эВ - 124 кэВ",
					Frequency:   "30 ПГц - 30 ЭГц",
					Intensity:   5.0,
					ImageURL:    "http://localhost:9000/physicsservice/xray.jpg", // Измените на URL из MinIO
				},
			},
			request: []RequestItem{},
		}
	})
	return instance
}

func (r *Repository) GetServices() []Service {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	services := make([]Service, 0, len(r.services))
	for _, s := range r.services {
		services = append(services, s)
	}
	return services
}

func (r *Repository) GetServiceByID(id int) (Service, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	service, exists := r.services[id]
	if !exists {
		return Service{}, errors.New("service not found")
	}
	return service, nil
}

func (r *Repository) AddToRequest(item RequestItem) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if item already exists
	for i, existingItem := range r.request {
		if existingItem.ServiceID == item.ServiceID {
			r.request[i] = item
			return
		}
	}

	// If not exists, add new
	r.request = append(r.request, item)
}

func (r *Repository) UpdateRequestItem(updatedItem RequestItem) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, item := range r.request {
		if item.ServiceID == updatedItem.ServiceID {
			r.request[i] = updatedItem
			return
		}
	}
}

func (r *Repository) RemoveFromRequest(serviceID int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, item := range r.request {
		if item.ServiceID == serviceID {
			r.request = append(r.request[:i], r.request[i+1:]...)
			return
		}
	}
}

func (r *Repository) GetRequest() []RequestItem {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	requestCopy := make([]RequestItem, len(r.request))
	copy(requestCopy, r.request)
	return requestCopy
}

func (r *Repository) GetRequestCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.request)
}
