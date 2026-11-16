package domain_test

import (
	"errors"
	"math"
	"testing"

	"pos/internal/domain"
)

func TestNewStudent(t *testing.T) {
	s, err := domain.NewStudent(1, "  Иванов   Иван ")
	if err != nil {
		t.Fatal(err)
	}
	if s.ID() != 1 || s.Name() != "Иванов Иван" {
		t.Errorf("получили #%d %q", s.ID(), s.Name())
	}

	if _, err := domain.NewStudent(1, "   "); !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("пустое ФИО: %v", err)
	}
	if _, err := domain.NewStudent(0, "Иванов Иван"); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("номер 0: %v", err)
	}
}

func TestStudentIsEntity(t *testing.T) {
	a, _ := domain.NewStudent(1, "Иванов Иван")
	b, _ := domain.NewStudent(2, "Иванов Иван")
	if a == b {
		t.Error("тёзки с разными номерами - разные студенты")
	}
}

func TestNewScore(t *testing.T) {
	for _, points := range []float64{0, 7.5, 10} {
		s, err := domain.NewScore(points, 10)
		if err != nil {
			t.Errorf("%v из 10: %v", points, err)
			continue
		}
		if s.Points() != points || s.Max() != 10 {
			t.Errorf("получили %v", s)
		}
	}

	for _, points := range []float64{-0.5, 10.01, 12, math.NaN(), math.Inf(1)} {
		if _, err := domain.NewScore(points, 10); !errors.Is(err, domain.ErrScoreOutOfRange) {
			t.Errorf("%v из 10: ожидали ErrScoreOutOfRange, получили %v", points, err)
		}
	}
	for _, maxScore := range []float64{0, -10, math.NaN()} {
		if _, err := domain.NewScore(0, maxScore); !errors.Is(err, domain.ErrInvalidMaxScore) {
			t.Errorf("максимум %v: %v", maxScore, err)
		}
	}
}

func TestScoreIsValueObject(t *testing.T) {
	a, _ := domain.NewScore(8, 10)
	b, _ := domain.NewScore(8, 10)
	if a != b {
		t.Error(`два "8 из 10" должны быть равны`)
	}
	if a.Ratio() != 0.8 || a.String() != "8 из 10" {
		t.Errorf("Ratio %v, String %q", a.Ratio(), a)
	}
}

func TestNewCriterion(t *testing.T) {
	c, err := domain.NewCriterion(1, " Домашние  задания ", 10, 0.4)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID() != 1 || c.Name() != "Домашние задания" || c.MaxScore() != 10 || c.Weight() != 0.4 {
		t.Errorf("получили %+v", c)
	}

	tests := []struct {
		name     string
		id       domain.CriterionID
		title    string
		maxScore float64
		weight   float64
		want     error
	}{
		{"номер 0", 0, "ДЗ", 10, 1, domain.ErrInvalidID},
		{"пустое название", 1, " ", 10, 1, domain.ErrEmptyName},
		{"максимум 0", 1, "ДЗ", 0, 1, domain.ErrInvalidMaxScore},
		{"максимум бесконечность", 1, "ДЗ", math.Inf(1), 1, domain.ErrInvalidMaxScore},
		{"вес 0", 1, "ДЗ", 10, 0, domain.ErrInvalidWeight},
		{"вес отрицательный", 1, "ДЗ", 10, -0.5, domain.ErrInvalidWeight},
		{"вес NaN", 1, "ДЗ", 10, math.NaN(), domain.ErrInvalidWeight},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewCriterion(tt.id, tt.title, tt.maxScore, tt.weight)
			if !errors.Is(err, tt.want) {
				t.Errorf("ожидали %v, получили %v", tt.want, err)
			}
		})
	}
}

func TestCriterionScore(t *testing.T) {
	c, _ := domain.NewCriterion(1, "Контрольная", 5, 1)
	if s, err := c.Score(5); err != nil || s.Max() != 5 {
		t.Errorf("5 из 5: %v, %v", s, err)
	}
	_, err := c.Score(12)
	if !errors.Is(err, domain.ErrScoreOutOfRange) {
		t.Fatalf("12 из 5: %v", err)
	}
	if want := `критерий "Контрольная": балл выходит за пределы критерия: 12, можно от 0 до 5`; err.Error() != want {
		t.Errorf("сообщение:\n%s\nожидали:\n%s", err, want)
	}
}
