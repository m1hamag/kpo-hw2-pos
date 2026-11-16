package domain

import (
	"fmt"
	"strings"
	"time"
)

// EvaluationKey показывает, кому и по какому критерию поставили оценку
type EvaluationKey struct {
	Student   StudentID
	Criterion CriterionID
}

// Evaluation хранит оценку студента по критерию. Создать её можно только через EvaluationFactory
type Evaluation struct {
	key     EvaluationKey
	score   Score
	date    time.Time
	comment string
}

// Key возвращает студента и критерий оценки
func (e Evaluation) Key() EvaluationKey { return e.key }

// Student возвращает номер студента, которому поставили оценку
func (e Evaluation) Student() StudentID { return e.key.Student }

// Criterion возвращает номер критерия
func (e Evaluation) Criterion() CriterionID { return e.key.Criterion }

// Score возвращает балл
func (e Evaluation) Score() Score { return e.score }

// Date возвращает день, когда оценку поставили или последний раз исправили
func (e Evaluation) Date() time.Time { return e.date }

// Comment возвращает комментарий преподавателя, он может быть пустым
func (e Evaluation) Comment() string { return e.comment }

// EvaluationFactory это единственное место, где создаются оценки (паттерн "Фабрика")
type EvaluationFactory struct {
	rubric *Rubric
}

// NewEvaluationFactory создаёт фабрику для текущего рубрикатора
func NewEvaluationFactory(rubric *Rubric) EvaluationFactory {
	return EvaluationFactory{rubric: rubric}
}

// Create создаёт оценку, если критерий есть в рубрикаторе и балл не выходит
// за его пределы. От даты остаётся только день
func (f EvaluationFactory) Create(student StudentID, criterion CriterionID, points float64,
	date time.Time, comment string) (Evaluation, error) {
	if student <= 0 {
		return Evaluation{}, fmt.Errorf("студент: %w (%d)", ErrInvalidID, student)
	}
	if f.rubric == nil {
		return Evaluation{}, fmt.Errorf("%w: №%d", ErrUnknownCriterion, criterion)
	}
	c, err := f.rubric.Criterion(criterion)
	if err != nil {
		return Evaluation{}, err
	}
	score, err := c.Score(points)
	if err != nil {
		return Evaluation{}, err
	}
	if date.IsZero() {
		return Evaluation{}, ErrNoDate
	}
	y, m, d := date.Date()
	return Evaluation{
		key:     EvaluationKey{Student: student, Criterion: criterion},
		score:   score,
		date:    time.Date(y, m, d, 0, 0, 0, 0, time.UTC),
		comment: strings.TrimSpace(comment),
	}, nil
}
