package domain

import (
	"fmt"
	"slices"
	"strings"
)

// Rubric хранит критерии, по которым оценивают на курсе
type Rubric struct {
	criteria []Criterion // в порядке добавления
}

// NewRubric собирает рубрикатор из готовых критериев. Без аргументов получится пустой
func NewRubric(criteria ...Criterion) (*Rubric, error) {
	r := &Rubric{}
	for _, c := range criteria {
		if err := r.add(c); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// AddCriterion добавляет новый критерий и сам выдаёт ему номер
func (r *Rubric) AddCriterion(name string, maxScore, weight float64) (Criterion, error) {
	c, err := NewCriterion(r.nextID(), name, maxScore, weight)
	if err != nil {
		return Criterion{}, err
	}
	if err := r.add(c); err != nil {
		return Criterion{}, err
	}
	return c, nil
}

// Criterion находит критерий по номеру. Если его нет, вернёт ErrUnknownCriterion
func (r *Rubric) Criterion(id CriterionID) (Criterion, error) {
	if i := r.index(id); i >= 0 {
		return r.criteria[i], nil
	}
	return Criterion{}, fmt.Errorf("%w: №%d", ErrUnknownCriterion, id)
}

// Criteria возвращает копию списка критериев в порядке добавления
func (r *Rubric) Criteria() []Criterion {
	return slices.Clone(r.criteria)
}

// IsEmpty сообщает, что критериев пока нет
func (r *Rubric) IsEmpty() bool { return len(r.criteria) == 0 }

// RemoveCriterion убирает критерий из рубрикатора
func (r *Rubric) RemoveCriterion(id CriterionID) error {
	i := r.index(id)
	if i < 0 {
		return fmt.Errorf("%w: №%d", ErrUnknownCriterion, id)
	}
	r.criteria = slices.Delete(r.criteria, i, i+1)
	return nil
}

func (r *Rubric) add(c Criterion) error {
	if c.id <= 0 {
		return fmt.Errorf("критерий: %w (%d)", ErrInvalidID, c.id)
	}
	for _, other := range r.criteria {
		if other.id == c.id {
			return fmt.Errorf("%w: номер %d занят", ErrDuplicateCriterion, c.id)
		}
		if strings.EqualFold(other.name, c.name) {
			return fmt.Errorf("%w: %q", ErrDuplicateCriterion, other.name)
		}
	}
	r.criteria = append(r.criteria, c)
	return nil
}

func (r *Rubric) index(id CriterionID) int {
	return slices.IndexFunc(r.criteria, func(c Criterion) bool { return c.id == id })
}

func (r *Rubric) nextID() CriterionID {
	var maxID CriterionID
	for _, c := range r.criteria {
		maxID = max(maxID, c.id)
	}
	return maxID + 1
}
