// Package domain содержит ядро ПОС со студентами, рубрикатором, оценками и итогом
package domain

import (
	"errors"
	"math"
	"strconv"
)

// Ошибки домена. К ним дописываются подробности, поэтому сравнивайте через errors.Is
var (
	ErrInvalidID          = errors.New("номер должен быть положительным")
	ErrEmptyName          = errors.New("не может быть пустым")
	ErrInvalidMaxScore    = errors.New("максимум критерия должен быть больше 0")
	ErrInvalidWeight      = errors.New("вес критерия должен быть больше 0")
	ErrDuplicateCriterion = errors.New("такой критерий уже есть в рубрикаторе")
	ErrUnknownCriterion   = errors.New("критерия нет в рубрикаторе")
	ErrScoreOutOfRange    = errors.New("балл выходит за пределы критерия")
	ErrNoDate             = errors.New("у оценки должна быть дата")
	ErrStudentNotFound    = errors.New("студент не найден")
	ErrEvaluationNotFound = errors.New("оценка не найдена")

	ErrEmptyRubric         = errors.New("в рубрикаторе нет ни одного критерия")
	ErrForeignEvaluation   = errors.New("оценка поставлена другому студенту")
	ErrDuplicateEvaluation = errors.New("по критерию больше одной оценки")
	ErrFinalOutOfRange     = errors.New("итог вышел за пределы от 0 до 10")
)

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

// formatNumber печатает число без лишних нулей, например 10, 7.5 или 0.25
func formatNumber(x float64) string {
	return strconv.FormatFloat(x, 'f', -1, 64)
}
