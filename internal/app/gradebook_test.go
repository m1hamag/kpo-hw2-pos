package app_test

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"pos/internal/adapters/secondary/jsonstore"
	"pos/internal/app"
	"pos/internal/domain"
)

var today = time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC)

// spyExporter запоминает, что ему передали на выгрузку
type spyExporter struct {
	got []app.StudentFinal
	err error
}

func (s *spyExporter) Export(w io.Writer, finals []app.StudentFinal) error {
	s.got = finals
	return s.err
}

// newGradebook собирает фасад на хранилище в памяти, как это сделает DI-контейнер
func newGradebook(t *testing.T, exporter app.FinalsExporter) *app.Gradebook {
	t.Helper()
	store, err := jsonstore.Open("")
	if err != nil {
		t.Fatal(err)
	}
	if exporter == nil {
		exporter = &spyExporter{}
	}
	return app.New(store.Students(), store.Rubric(), store.Evaluations(), exporter,
		[]domain.FinalPolicy{domain.WeightedPolicy{}, domain.GradedOnlyPolicy{}},
		func() time.Time { return today })
}

// course заводит двух студентов и три критерия
func course(t *testing.T, g *app.Gradebook) {
	t.Helper()
	for _, name := range []string{"Иванов Иван", "Петрова Анна"} {
		if _, err := g.AddStudent(name); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		name        string
		max, weight float64
	}{{"ДЗ", 10, 0.4}, {"Контрольная", 5, 0.2}, {"Экзамен", 10, 0.4}} {
		if _, err := g.AddCriterion(c.name, c.max, c.weight); err != nil {
			t.Fatal(err)
		}
	}
}

// must достаёт значение и роняет тест, если сценарий вернул ошибку
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestScenario_StudentsCriteriaGradesFinalExport(t *testing.T) {
	spy := &spyExporter{}
	g := newGradebook(t, spy)
	course(t, g)

	students := must(g.Students())
	if len(students) != 2 || students[0].ID() != 1 || students[1].Name() != "Петрова Анна" {
		t.Fatalf("студенты %v", students)
	}
	if criteria := must(g.Criteria()); len(criteria) != 3 {
		t.Fatalf("критериев %d", len(criteria))
	}

	e := must(g.Grade(1, 1, 8, " хорошо "))
	if e.Score().String() != "8 из 10" || !e.Date().Equal(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)) || e.Comment() != "хорошо" {
		t.Errorf("оценка %v, %v, %q", e.Score(), e.Date(), e.Comment())
	}
	must(g.Grade(1, 2, 5, ""))
	must(g.Grade(1, 3, 6, ""))
	must(g.Grade(2, 1, 10, ""))

	f := must(g.Final(1))
	if f.Student.Name() != "Иванов Иван" || f.Final.Value() != 7.6 || f.Final.Rounded() != 8 {
		t.Errorf("итог Иванова %v (%d)", f.Final.Value(), f.Final.Rounded())
	}

	finals := must(g.Finals())
	if len(finals) != 2 || finals[1].Final.Value() != 4 {
		t.Errorf("итоги %v", finals)
	}

	n := must(g.ExportFinals(&bytes.Buffer{}))
	if n != 2 || len(spy.got) != 2 || spy.got[0].Final.Value() != 7.6 {
		t.Errorf("выгружено %d, экспорт получил %v", n, spy.got)
	}
}

func TestGrade_OnlyOnceThenCorrect(t *testing.T) {
	g := newGradebook(t, nil)
	course(t, g)
	must(g.Grade(1, 1, 6, ""))

	if _, err := g.Grade(1, 1, 9, ""); !errors.Is(err, app.ErrAlreadyEvaluated) {
		t.Errorf("повторная оценка: %v", err)
	}
	fixed := must(g.Correct(1, 1, 9, "пересдача"))
	if fixed.Score().Points() != 9 || fixed.Comment() != "пересдача" {
		t.Errorf("исправленная оценка %v %q", fixed.Score(), fixed.Comment())
	}
	if _, err := g.Correct(1, 2, 3, ""); !errors.Is(err, domain.ErrEvaluationNotFound) {
		t.Errorf("исправление несуществующей: %v", err)
	}

	if err := g.Cancel(1, 1); err != nil {
		t.Fatal(err)
	}
	if err := g.Cancel(1, 1); !errors.Is(err, domain.ErrEvaluationNotFound) {
		t.Errorf("повторная отмена: %v", err)
	}
	must(g.Grade(1, 1, 7, ""))
}

func TestGrade_DomainRulesHold(t *testing.T) {
	g := newGradebook(t, nil)
	course(t, g)
	tests := []struct {
		name      string
		student   domain.StudentID
		criterion domain.CriterionID
		points    float64
		want      error
	}{
		{"12 из 10", 1, 1, 12, domain.ErrScoreOutOfRange},
		{"6 из 5", 1, 2, 6, domain.ErrScoreOutOfRange},
		{"критерия нет в рубрикаторе", 1, 9, 5, domain.ErrUnknownCriterion},
		{"студента нет", 9, 1, 5, domain.ErrStudentNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := g.Grade(tt.student, tt.criterion, tt.points, ""); !errors.Is(err, tt.want) {
				t.Errorf("ожидали %v, получили %v", tt.want, err)
			}
		})
	}
	if f := must(g.Final(1)); f.Final.Graded() != 0 {
		t.Error("после отказов оценок быть не должно")
	}

	must(g.Grade(1, 2, 4, ""))
	if _, err := g.Correct(1, 2, 7, ""); !errors.Is(err, domain.ErrScoreOutOfRange) {
		t.Errorf("исправление на 7 из 5: %v", err)
	}
	if f := must(g.Final(1)); f.Final.Lines()[1].Evaluation.Score().Points() != 4 {
		t.Error("неудачное исправление испортило оценку")
	}
}

func TestFinal_FollowsCurrentEvaluations(t *testing.T) {
	g := newGradebook(t, nil)
	course(t, g)
	must(g.Grade(1, 1, 10, ""))
	if v := must(g.Final(1)).Final.Value(); v != 4 {
		t.Errorf("после оценки итог %v, ожидали 4", v)
	}
	must(g.Correct(1, 1, 5, ""))
	if v := must(g.Final(1)).Final.Value(); v != 2 {
		t.Errorf("после исправления итог %v, ожидали 2", v)
	}
	g.Cancel(1, 1)
	if v := must(g.Final(1)).Final.Value(); v != 0 {
		t.Errorf("после отмены итог %v, ожидали 0", v)
	}
}

func TestUsePolicy(t *testing.T) {
	g := newGradebook(t, nil)
	course(t, g)
	must(g.Grade(1, 1, 10, ""))

	if len(g.Policies()) != 2 || g.Policy().Name() != (domain.WeightedPolicy{}).Name() {
		t.Fatalf("правила %v, текущее %q", g.Policies(), g.Policy().Name())
	}
	if err := g.UsePolicy(1); err != nil {
		t.Fatal(err)
	}
	f := must(g.Final(1))
	if f.Final.Value() != 10 || f.Final.Policy() != (domain.GradedOnlyPolicy{}).Name() {
		t.Errorf("по выставленным итог %v по правилу %q", f.Final.Value(), f.Final.Policy())
	}
	for _, i := range []int{-1, 2} {
		if err := g.UsePolicy(i); !errors.Is(err, app.ErrUnknownPolicy) {
			t.Errorf("UsePolicy(%d): %v", i, err)
		}
	}
	if g.Policy().Name() != (domain.GradedOnlyPolicy{}).Name() {
		t.Error("неверный номер не должен сбрасывать правило")
	}
}

func TestRemoveCriterion(t *testing.T) {
	g := newGradebook(t, nil)
	course(t, g)
	must(g.Grade(1, 2, 3, ""))
	must(g.Grade(2, 2, 4, ""))

	if err := g.RemoveCriterion(2); !errors.Is(err, app.ErrCriterionInUse) {
		t.Errorf("удаление критерия с оценками: %v", err)
	}
	g.Cancel(1, 2)
	g.Cancel(2, 2)
	if err := g.RemoveCriterion(2); err != nil {
		t.Fatal(err)
	}
	if criteria := must(g.Criteria()); len(criteria) != 2 {
		t.Errorf("критериев %d, ожидали 2", len(criteria))
	}
	if err := g.RemoveCriterion(2); !errors.Is(err, domain.ErrUnknownCriterion) {
		t.Errorf("повторное удаление: %v", err)
	}
}

func TestRemoveStudent_WithEvaluations(t *testing.T) {
	g := newGradebook(t, nil)
	course(t, g)
	must(g.Grade(1, 1, 8, ""))
	must(g.Grade(2, 1, 9, ""))

	if err := g.RemoveStudent(1); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Final(1); !errors.Is(err, domain.ErrStudentNotFound) {
		t.Errorf("итог удалённого: %v", err)
	}
	if f := must(g.Final(2)); f.Final.Graded() != 1 {
		t.Error("удаление задело оценки другого студента")
	}
	if err := g.RemoveStudent(1); !errors.Is(err, domain.ErrStudentNotFound) {
		t.Errorf("повторное удаление: %v", err)
	}
	if id := must(g.AddStudent("Сидоров Пётр")).ID(); id != 3 {
		t.Errorf("новый студент получил №%d", id)
	}
}

func TestAddStudentAndCriterion_Errors(t *testing.T) {
	g := newGradebook(t, nil)
	if _, err := g.AddStudent("  "); !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("пустое ФИО: %v", err)
	}
	must(g.AddCriterion("ДЗ", 10, 1))
	if _, err := g.AddCriterion("дз", 5, 1); !errors.Is(err, domain.ErrDuplicateCriterion) {
		t.Errorf("повтор критерия: %v", err)
	}
	if _, err := g.AddCriterion("Экзамен", 10, 0); !errors.Is(err, domain.ErrInvalidWeight) {
		t.Errorf("вес 0: %v", err)
	}
}

func TestFinals_EmptyRubric(t *testing.T) {
	g := newGradebook(t, nil)
	if finals := must(g.Finals()); len(finals) != 0 {
		t.Errorf("без студентов итогов %d", len(finals))
	}
	must(g.AddStudent("Иванов Иван"))
	if _, err := g.Finals(); !errors.Is(err, domain.ErrEmptyRubric) {
		t.Errorf("без критериев: %v", err)
	}
	if _, err := g.ExportFinals(io.Discard); !errors.Is(err, domain.ErrEmptyRubric) {
		t.Errorf("выгрузка без критериев: %v", err)
	}
}

func TestExportFinals_ExporterError(t *testing.T) {
	boom := errors.New("диск переполнен")
	g := newGradebook(t, &spyExporter{err: boom})
	course(t, g)
	if _, err := g.ExportFinals(io.Discard); !errors.Is(err, boom) {
		t.Errorf("ошибка экспорта потерялась: %v", err)
	}
}

func TestNew_Defaults(t *testing.T) {
	store, _ := jsonstore.Open("")
	g := app.New(store.Students(), store.Rubric(), store.Evaluations(), &spyExporter{}, nil, nil)
	if len(g.Policies()) != 1 || g.Policy().Name() != (domain.WeightedPolicy{}).Name() {
		t.Errorf("правила по умолчанию %v", g.Policies())
	}
	g.AddStudent("Иванов Иван")
	g.AddCriterion("ДЗ", 10, 1)
	e := must(g.Grade(1, 1, 5, ""))
	if e.Date().IsZero() {
		t.Error("без часов должна ставиться сегодняшняя дата")
	}
}
