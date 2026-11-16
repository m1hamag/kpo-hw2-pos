// Package app содержит сценарии ПОС. Консоль вызывает только фасад Gradebook
package app

import (
	"errors"
	"io"
	"time"

	"pos/internal/domain"
)

// Ошибки сценариев
var (
	ErrAlreadyEvaluated = errors.New("оценка по этому критерию уже стоит, её можно исправить")
	ErrCriterionInUse   = errors.New("по критерию уже стоят оценки, сначала отмените их")
	ErrUnknownPolicy    = errors.New("нет такого правила подсчёта")
)

// StudentFinal хранит итог одного студента для ведомости и выгрузки
type StudentFinal struct {
	Student domain.Student
	Final   domain.FinalGrade
}

// FinalsExporter выгружает итоги. Это порт, его реализует адаптер csvexport
type FinalsExporter interface {
	Export(w io.Writer, finals []StudentFinal) error
}

// Clock возвращает текущее время, в тестах его подменяют
type Clock func() time.Time
