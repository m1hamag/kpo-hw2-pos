package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"pos/internal/domain"
)

func TestRubric_AddCriterion(t *testing.T) {
	r, _ := domain.NewRubric()
	if !r.IsEmpty() {
		t.Fatal("новый рубрикатор должен быть пустым")
	}
	hw, err := r.AddCriterion("ДЗ", 10, 0.4)
	if err != nil {
		t.Fatal(err)
	}
	test, _ := r.AddCriterion("Контрольная", 5, 0.2)
	if hw.ID() != 1 || test.ID() != 2 {
		t.Errorf("номера %d, %d - ожидали 1, 2", hw.ID(), test.ID())
	}

	if _, err := r.AddCriterion("дз", 100, 1); !errors.Is(err, domain.ErrDuplicateCriterion) {
		t.Errorf("повтор названия без учёта регистра: %v", err)
	}
	if _, err := r.AddCriterion("Экзамен", 0, 1); !errors.Is(err, domain.ErrInvalidMaxScore) {
		t.Errorf("максимум 0: %v", err)
	}
	if got := len(r.Criteria()); got != 2 {
		t.Errorf("после отказов критериев %d, ожидали 2", got)
	}

	got, err := r.Criterion(2)
	if err != nil || got != test {
		t.Errorf("Criterion(2) = %+v, %v", got, err)
	}
	if _, err := r.Criterion(42); !errors.Is(err, domain.ErrUnknownCriterion) {
		t.Errorf("Criterion(42): %v", err)
	}
}

func TestRubric_CriteriaIsCopy(t *testing.T) {
	r, _ := domain.NewRubric()
	r.AddCriterion("ДЗ", 10, 1)
	list := r.Criteria()
	list[0] = domain.Criterion{}
	if c, err := r.Criterion(1); err != nil || c.Name() != "ДЗ" {
		t.Error("изменение копии испортило рубрикатор")
	}
}

func TestRubric_RemoveCriterion(t *testing.T) {
	r, _ := domain.NewRubric()
	r.AddCriterion("ДЗ", 10, 1)
	r.AddCriterion("Контрольная", 5, 1)
	r.AddCriterion("Экзамен", 10, 1)

	if err := r.RemoveCriterion(2); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Criterion(2); !errors.Is(err, domain.ErrUnknownCriterion) {
		t.Errorf("удалённый критерий найден: %v", err)
	}
	if err := r.RemoveCriterion(2); !errors.Is(err, domain.ErrUnknownCriterion) {
		t.Errorf("повторное удаление: %v", err)
	}
	if c, _ := r.AddCriterion("Проект", 10, 1); c.ID() != 4 {
		t.Errorf("новый критерий получил номер %d, ожидали 4", c.ID())
	}
	var names []string
	for _, c := range r.Criteria() {
		names = append(names, c.Name())
	}
	if want := "[ДЗ Экзамен Проект]"; fmt.Sprint(names) != want {
		t.Errorf("порядок %v, ожидали %s", names, want)
	}
}

func TestNewRubric_Restore(t *testing.T) {
	hw, _ := domain.NewCriterion(3, "ДЗ", 10, 1)
	exam, _ := domain.NewCriterion(7, "Экзамен", 10, 1)
	r, err := domain.NewRubric(hw, exam)
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := r.AddCriterion("Проект", 5, 1); c.ID() != 8 {
		t.Errorf("после восстановления номер %d, ожидали 8", c.ID())
	}

	sameID, _ := domain.NewCriterion(3, "Другое", 10, 1)
	if _, err := domain.NewRubric(hw, sameID); !errors.Is(err, domain.ErrDuplicateCriterion) {
		t.Errorf("повтор номера: %v", err)
	}
	if _, err := domain.NewRubric(domain.Criterion{}); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("пустой критерий: %v", err)
	}
}
