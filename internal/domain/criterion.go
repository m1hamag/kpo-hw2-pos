package domain

import (
	"fmt"
	"strings"
)

// CriterionID это номер критерия в рубрикаторе
type CriterionID int

// Criterion описывает критерий оценивания с его максимумом и весом в итоге
type Criterion struct {
	id       CriterionID
	name     string
	maxScore float64
	weight   float64
}

// NewCriterion создаёт критерий и проверяет номер, название, максимум и вес
func NewCriterion(id CriterionID, name string, maxScore, weight float64) (Criterion, error) {
	if id <= 0 {
		return Criterion{}, fmt.Errorf("критерий: %w (%d)", ErrInvalidID, id)
	}
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return Criterion{}, fmt.Errorf("название критерия %w", ErrEmptyName)
	}
	if !isFinite(maxScore) || maxScore <= 0 {
		return Criterion{}, fmt.Errorf("%w (%s)", ErrInvalidMaxScore, formatNumber(maxScore))
	}
	if !isFinite(weight) || weight <= 0 {
		return Criterion{}, fmt.Errorf("%w (%s)", ErrInvalidWeight, formatNumber(weight))
	}
	return Criterion{id: id, name: name, maxScore: maxScore, weight: weight}, nil
}

// ID возвращает номер критерия
func (c Criterion) ID() CriterionID { return c.id }

// Name возвращает название критерия
func (c Criterion) Name() string { return c.name }

// MaxScore возвращает максимальный балл
func (c Criterion) MaxScore() float64 { return c.maxScore }

// Weight возвращает вес критерия в итоге
func (c Criterion) Weight() float64 { return c.weight }

// Score проверяет баллы по этому критерию и возвращает ошибку,
// если они меньше 0 или больше максимума
func (c Criterion) Score(points float64) (Score, error) {
	s, err := NewScore(points, c.maxScore)
	if err != nil {
		return Score{}, fmt.Errorf("критерий %q: %w", c.name, err)
	}
	return s, nil
}
