package csvexport_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"pos/internal/adapters/secondary/csvexport"
	"pos/internal/app"
	"pos/internal/domain"
)

func TestExport(t *testing.T) {
	r, _ := domain.NewRubric()
	r.AddCriterion("ДЗ", 10, 0.4)
	r.AddCriterion("Контрольная", 5, 0.2)
	r.AddCriterion("Экзамен", 10, 0.4)
	f := domain.NewEvaluationFactory(r)
	day := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	ev := func(s domain.StudentID, c domain.CriterionID, p float64) domain.Evaluation {
		e, err := f.Create(s, c, p, day, "")
		if err != nil {
			t.Fatal(err)
		}
		return e
	}

	ivanov, _ := domain.NewStudent(1, "Иванов Иван")
	petrova, _ := domain.NewStudent(2, "Петрова Анна")
	service := domain.NewFinalGradeService(domain.WeightedPolicy{})
	f1, _ := service.Final(1, r, []domain.Evaluation{ev(1, 1, 8), ev(1, 2, 5), ev(1, 3, 6)})
	f2, _ := service.Final(2, r, []domain.Evaluation{ev(2, 1, 9.5)})

	var buf bytes.Buffer
	err := csvexport.New().Export(&buf, []app.StudentFinal{{Student: ivanov, Final: f1}, {Student: petrova, Final: f2}})
	if err != nil {
		t.Fatal(err)
	}
	want := "№;Студент;ДЗ (из 10);Контрольная (из 5);Экзамен (из 10);Итог;Оценка;Правило\n" +
		"1;Иванов Иван;8;5;6;7,60;8;по всем критериям, без оценки 0 баллов\n" +
		"2;Петрова Анна;9,5;;;3,80;4;по всем критериям, без оценки 0 баллов\n"
	if buf.String() != want {
		t.Errorf("получили:\n%s\nожидали:\n%s", buf.String(), want)
	}
}

func TestExport_NoStudents(t *testing.T) {
	var buf bytes.Buffer
	if err := csvexport.New().Export(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if want := "№;Студент;Итог;Оценка;Правило\n"; buf.String() != want {
		t.Errorf("получили %q", buf.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("диск переполнен")
}

func TestExport_WriteError(t *testing.T) {
	if err := csvexport.New().Export(failingWriter{}, nil); err == nil {
		t.Error("ошибка записи потерялась")
	}
}
