package domain_test

import (
	"errors"
	"math"
	"testing"

	"pos/internal/domain"
)

// grade ставит оценку через фабрику и роняет тест при ошибке
func grade(t *testing.T, r *domain.Rubric, student domain.StudentID, criterion domain.CriterionID, points float64) domain.Evaluation {
	t.Helper()
	e, err := domain.NewEvaluationFactory(r).Create(student, criterion, points, day, "")
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// fixedPolicy всегда возвращает одно и то же число, чтобы проверить сервис отдельно от формулы
type fixedPolicy float64

func (fixedPolicy) Name() string                         { return "фиксированное" }
func (p fixedPolicy) Compute([]domain.FinalLine) float64 { return float64(p) }

func TestFinal_WeightedPolicy(t *testing.T) {
	// ДЗ 10 баллов с весом 0.4, контрольная 5 баллов с весом 0.2, экзамен 10 баллов с весом 0.4
	r := newRubric(t)
	s := domain.NewFinalGradeService(domain.WeightedPolicy{})
	evals := []domain.Evaluation{grade(t, r, 1, 1, 8), grade(t, r, 1, 2, 5)}

	g, err := s.Final(1, r, evals)
	if err != nil {
		t.Fatal(err)
	}
	// (0.4*0.8 + 0.2*1 + 0.4*0) / 1 * 10 = 5.2
	if g.Value() != 5.2 || g.Rounded() != 5 {
		t.Errorf("без экзамена итог %v (%d), ожидали 5.2 (5)", g.Value(), g.Rounded())
	}
	if g.Graded() != 2 || g.Complete() {
		t.Errorf("выставлено %d, полный %v", g.Graded(), g.Complete())
	}

	evals = append(evals, grade(t, r, 1, 3, 6))
	g, _ = s.Final(1, r, evals)
	// 0.32 + 0.2 + 0.24 = 0.76, то есть 7.6
	if g.Value() != 7.6 || g.Rounded() != 8 || !g.Complete() {
		t.Errorf("с экзаменом итог %v (%d), полный %v", g.Value(), g.Rounded(), g.Complete())
	}
	if g.Student() != 1 || g.Policy() != (domain.WeightedPolicy{}).Name() {
		t.Errorf("студент %d, правило %q", g.Student(), g.Policy())
	}
}

func TestFinal_GradedOnlyPolicy(t *testing.T) {
	r := newRubric(t)
	s := domain.NewFinalGradeService(domain.GradedOnlyPolicy{})

	g, err := s.Final(1, r, []domain.Evaluation{grade(t, r, 1, 1, 8), grade(t, r, 1, 2, 5)})
	if err != nil {
		t.Fatal(err)
	}
	// (0.32 + 0.2) / 0.6 * 10 = 8.666...
	if g.Value() != 8.67 || g.Rounded() != 9 {
		t.Errorf("итог %v (%d), ожидали 8.67 (9)", g.Value(), g.Rounded())
	}

	g, _ = s.Final(1, r, nil)
	if g.Value() != 0 || g.Graded() != 0 {
		t.Errorf("без оценок итог %v", g.Value())
	}
}

func TestFinal_StrategyChangesOnlyFormula(t *testing.T) {
	r := newRubric(t)
	evals := []domain.Evaluation{grade(t, r, 1, 1, 10)}
	all, _ := domain.NewFinalGradeService(domain.WeightedPolicy{}).Final(1, r, evals)
	graded, _ := domain.NewFinalGradeService(domain.GradedOnlyPolicy{}).Final(1, r, evals)
	if all.Value() != 4 || graded.Value() != 10 {
		t.Errorf("по всем %v, по выставленным %v, ожидали 4 и 10", all.Value(), graded.Value())
	}
	if len(all.Lines()) != len(graded.Lines()) {
		t.Error("строки итога не должны зависеть от правила")
	}
}

func TestFinal_LinesFollowRubric(t *testing.T) {
	r := newRubric(t)
	exam := grade(t, r, 1, 3, 9)
	hw := grade(t, r, 1, 1, 7)
	g, err := domain.NewFinalGradeService(nil).Final(1, r, []domain.Evaluation{exam, hw})
	if err != nil {
		t.Fatal(err)
	}
	lines := g.Lines()
	if len(lines) != 3 {
		t.Fatalf("строк %d, ожидали 3", len(lines))
	}
	for i, want := range []struct {
		criterion domain.CriterionID
		points    float64
		graded    bool
	}{{1, 7, true}, {2, 0, false}, {3, 9, true}} {
		l := lines[i]
		if l.Criterion.ID() != want.criterion || (l.Evaluation != nil) != want.graded {
			t.Errorf("строка %d: критерий %d, оценка %v", i, l.Criterion.ID(), l.Evaluation)
			continue
		}
		if want.graded && l.Evaluation.Score().Points() != want.points {
			t.Errorf("строка %d: балл %v", i, l.Evaluation.Score())
		}
	}
}

func TestFinal_ChecksEvaluations(t *testing.T) {
	r := newRubric(t)
	other, _ := domain.NewRubric()
	other.AddCriterion("ДЗ", 10, 1)
	other.AddCriterion("Контрольная", 10, 1)
	other.AddCriterion("Экзамен", 10, 1)
	other.AddCriterion("Проект", 10, 1)

	s := domain.NewFinalGradeService(domain.WeightedPolicy{})
	tests := []struct {
		name   string
		rubric *domain.Rubric
		evals  []domain.Evaluation
		want   error
	}{
		{"чужая оценка", r, []domain.Evaluation{grade(t, r, 2, 1, 5)}, domain.ErrForeignEvaluation},
		{"критерия нет в рубрикаторе", r, []domain.Evaluation{grade(t, other, 1, 4, 5)}, domain.ErrUnknownCriterion},
		{"две оценки по критерию", r, []domain.Evaluation{grade(t, r, 1, 1, 5), grade(t, r, 1, 1, 6)}, domain.ErrDuplicateEvaluation},
		{"пустой рубрикатор", mustRubric(t), nil, domain.ErrEmptyRubric},
		{"нет рубрикатора", nil, nil, domain.ErrEmptyRubric},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := s.Final(1, tt.rubric, tt.evals); !errors.Is(err, tt.want) {
				t.Errorf("ожидали %v, получили %v", tt.want, err)
			}
		})
	}
}

func TestFinal_PolicyCannotBreakScale(t *testing.T) {
	r := newRubric(t)
	for _, value := range []float64{12, -1, math.NaN()} {
		_, err := domain.NewFinalGradeService(fixedPolicy(value)).Final(1, r, nil)
		if !errors.Is(err, domain.ErrFinalOutOfRange) {
			t.Errorf("правило вернуло %v: %v", value, err)
		}
	}
}

func TestFinalGrade_Rounded(t *testing.T) {
	r := newRubric(t)
	tests := []struct {
		value   float64
		rounded int
	}{
		{0, 0}, {4.49, 4}, {4.5, 5}, {7.499, 7}, {7.5, 8}, {0.1 + 0.2 + 7.2 - 0.000000000000001, 8}, {10, 10},
	}
	for _, tt := range tests {
		g, err := domain.NewFinalGradeService(fixedPolicy(tt.value)).Final(1, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		if g.Rounded() != tt.rounded {
			t.Errorf("%v округлилось до %d, ожидали %d", tt.value, g.Rounded(), tt.rounded)
		}
	}
}

func TestNewFinalGradeService_DefaultPolicy(t *testing.T) {
	s := domain.NewFinalGradeService(nil)
	if _, ok := s.Policy().(domain.WeightedPolicy); !ok {
		t.Errorf("по умолчанию %T, ожидали WeightedPolicy", s.Policy())
	}
}

func mustRubric(t *testing.T) *domain.Rubric {
	t.Helper()
	r, err := domain.NewRubric()
	if err != nil {
		t.Fatal(err)
	}
	return r
}
