package domain

import "fmt"

// Score хранит, сколько баллов набрано по критерию и сколько было возможно
type Score struct {
	points float64
	max    float64
}

// NewScore создаёт балл и проверяет, что он не меньше 0 и не больше max
func NewScore(points, max float64) (Score, error) {
	if !isFinite(max) || max <= 0 {
		return Score{}, fmt.Errorf("%w (%s)", ErrInvalidMaxScore, formatNumber(max))
	}
	if !isFinite(points) || points < 0 || points > max {
		return Score{}, fmt.Errorf("%w: %s, можно от 0 до %s",
			ErrScoreOutOfRange, formatNumber(points), formatNumber(max))
	}
	return Score{points: points, max: max}, nil
}

// Points возвращает набранные баллы
func (s Score) Points() float64 { return s.points }

// Max возвращает максимум критерия
func (s Score) Max() float64 { return s.max }

// Ratio показывает, какую часть максимума набрал студент (от 0 до 1)
func (s Score) Ratio() float64 { return s.points / s.max }

// String печатает балл в виде "8 из 10"
func (s Score) String() string {
	return formatNumber(s.points) + " из " + formatNumber(s.max)
}
