package domain_test

import (
	"errors"
	"testing"
	"time"

	"pos/internal/domain"
)

var day = time.Date(2026, time.November, 16, 18, 30, 0, 0, time.Local)

func newRubric(t *testing.T) *domain.Rubric {
	t.Helper()
	r, _ := domain.NewRubric()
	for _, c := range []struct {
		name        string
		max, weight float64
	}{{"ДЗ", 10, 0.4}, {"Контрольная", 5, 0.2}, {"Экзамен", 10, 0.4}} {
		if _, err := r.AddCriterion(c.name, c.max, c.weight); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestEvaluationFactory_Create(t *testing.T) {
	f := domain.NewEvaluationFactory(newRubric(t))
	e, err := f.Create(1, 2, 4.5, day, "  молодец ")
	if err != nil {
		t.Fatal(err)
	}
	if e.Key() != (domain.EvaluationKey{Student: 1, Criterion: 2}) {
		t.Errorf("ключ %+v", e.Key())
	}
	if e.Student() != 1 || e.Criterion() != 2 {
		t.Errorf("студент %d, критерий %d", e.Student(), e.Criterion())
	}
	if e.Score().String() != "4.5 из 5" {
		t.Errorf("балл %v - максимум должен прийти из критерия", e.Score())
	}
	if want := time.Date(2026, time.November, 16, 0, 0, 0, 0, time.UTC); !e.Date().Equal(want) {
		t.Errorf("дата %v, ожидали только день %v", e.Date(), want)
	}
	if e.Comment() != "молодец" {
		t.Errorf("комментарий %q", e.Comment())
	}
}

func TestEvaluationFactory_Invariants(t *testing.T) {
	f := domain.NewEvaluationFactory(newRubric(t))
	tests := []struct {
		name      string
		student   domain.StudentID
		criterion domain.CriterionID
		points    float64
		date      time.Time
		want      error
	}{
		{"12 из 10", 1, 1, 12, day, domain.ErrScoreOutOfRange},
		{"отрицательный балл", 1, 1, -1, day, domain.ErrScoreOutOfRange},
		{"6 из 5", 1, 2, 6, day, domain.ErrScoreOutOfRange},
		{"критерия нет в рубрикаторе", 1, 42, 5, day, domain.ErrUnknownCriterion},
		{"студент 0", 0, 1, 5, day, domain.ErrInvalidID},
		{"без даты", 1, 1, 5, time.Time{}, domain.ErrNoDate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.Create(tt.student, tt.criterion, tt.points, tt.date, "")
			if !errors.Is(err, tt.want) {
				t.Errorf("ожидали %v, получили %v", tt.want, err)
			}
		})
	}
}

func TestEvaluationFactory_NoRubric(t *testing.T) {
	_, err := domain.NewEvaluationFactory(nil).Create(1, 1, 5, day, "")
	if !errors.Is(err, domain.ErrUnknownCriterion) {
		t.Errorf("без рубрикатора: %v", err)
	}
}

func TestEvaluation_CorrectionKeepsIdentity(t *testing.T) {
	f := domain.NewEvaluationFactory(newRubric(t))
	first, _ := f.Create(1, 1, 6, day, "")
	fixed, _ := f.Create(1, 1, 9, day.AddDate(0, 0, 1), "пересдача")
	if first.Key() != fixed.Key() {
		t.Error("исправленная оценка должна остаться той же оценкой")
	}
	if first.Score() == fixed.Score() {
		t.Error("балл должен поменяться")
	}
}
