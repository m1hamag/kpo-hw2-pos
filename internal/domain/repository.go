package domain

// Порты хранения. Реализации лежат в адаптерах

// StudentRepository хранит студентов
type StudentRepository interface {
	// NextID выдаёт номер для нового студента
	NextID() (StudentID, error)
	// Save добавляет студента или заменяет студента с тем же номером
	Save(Student) error
	// Get находит студента. Если его нет, вернёт ErrStudentNotFound
	Get(StudentID) (Student, error)
	// List возвращает всех студентов по возрастанию номера
	List() ([]Student, error)
	// Delete удаляет студента. Если его нет, вернёт ErrStudentNotFound
	Delete(StudentID) error
}

// RubricRepository хранит рубрикатор курса (пока курс один)
type RubricRepository interface {
	// Load загружает рубрикатор. Если критериев ещё нет, он будет пустым
	Load() (*Rubric, error)
	// Save сохраняет рубрикатор целиком
	Save(*Rubric) error
}

// EvaluationRepository хранит оценки
type EvaluationRepository interface {
	// Save добавляет оценку или заменяет оценку с тем же ключом
	Save(Evaluation) error
	// Get находит оценку. Если её нет, вернёт ErrEvaluationNotFound
	Get(EvaluationKey) (Evaluation, error)
	// ByStudent возвращает оценки студента по возрастанию номера критерия
	ByStudent(StudentID) ([]Evaluation, error)
	// ByCriterion возвращает оценки по критерию по возрастанию номера студента
	ByCriterion(CriterionID) ([]Evaluation, error)
	// Delete удаляет оценку. Если её нет, вернёт ErrEvaluationNotFound
	Delete(EvaluationKey) error
	// DeleteByStudent удаляет все оценки студента
	DeleteByStudent(StudentID) error
}
