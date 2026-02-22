package repository

// DeleteRequestSQL - логическое удаление заявки через SQL UPDATE (требование README)
func (r *Repository) DeleteRequestSQL(requestID uint) error {
	result := r.db.Exec("UPDATE experiment_requests SET status = 'удалён' WHERE id = ?", requestID)
	return result.Error
}
