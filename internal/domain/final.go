package domain

import (
	"fmt"
	"math"
	"slices"
)

// MaxFinal это верхняя граница итога по десятибалльной шкале
const MaxFinal = 10.0

// FinalLine связывает критерий рубрикатора с оценкой студента по нему.
// Если оценки нет, Evaluation равен nil
type FinalLine struct {
	Criterion  Criterion
	Evaluation *Evaluation
}

// FinalGrade хранит итог студента вместе с оценками по каждому критерию.
// Итог нигде не сохраняется, его всегда считает FinalGradeService
type FinalGrade struct {
	student StudentID
	value   float64
	lines   []FinalLine
	policy  string
}

// Student возвращает номер студента
func (g FinalGrade) Student() StudentID { return g.student }

// Value возвращает итог, округлённый до сотых
func (g FinalGrade) Value() float64 { return math.Round(g.value*100) / 100 }

// Rounded округляет итог до целого, половина округляется вверх, как принято в ВШЭ
func (g FinalGrade) Rounded() int {
	// сначала убираем погрешность float, чтобы 7.4999999 считалось как 7.5
	cleaned := math.Round(g.value*1e6) / 1e6
	return int(math.Floor(cleaned + 0.5))
}

// Lines возвращает критерии с оценками в порядке рубрикатора
func (g FinalGrade) Lines() []FinalLine { return slices.Clone(g.lines) }

// Graded считает, по скольким критериям есть оценка
func (g FinalGrade) Graded() int {
	n := 0
	for _, l := range g.lines {
		if l.Evaluation != nil {
			n++
		}
	}
	return n
}

// Complete сообщает, что оценки стоят по всем критериям
func (g FinalGrade) Complete() bool { return g.Graded() == len(g.lines) }

// Policy возвращает название правила, по которому считали итог
func (g FinalGrade) Policy() string { return g.policy }

// FinalGradeService считает итог студента. Это доменный сервис, потому что
// итогу нужны сразу и рубрикатор, и все оценки студента
type FinalGradeService struct {
	policy FinalPolicy
}

// NewFinalGradeService создаёт сервис с правилом policy. Без правила берётся WeightedPolicy
func NewFinalGradeService(policy FinalPolicy) FinalGradeService {
	if policy == nil {
		policy = WeightedPolicy{}
	}
	return FinalGradeService{policy: policy}
}

// Policy возвращает правило, по которому сервис считает итог
func (s FinalGradeService) Policy() FinalPolicy { return s.policy }

// Final считает итог студента. Сначала проверяет, что все оценки принадлежат ему,
// поставлены по критериям из рубрикатора и не повторяются
func (s FinalGradeService) Final(student StudentID, rubric *Rubric, evaluations []Evaluation) (FinalGrade, error) {
	if rubric == nil || rubric.IsEmpty() {
		return FinalGrade{}, ErrEmptyRubric
	}
	byCriterion := make(map[CriterionID]Evaluation, len(evaluations))
	for _, e := range evaluations {
		if e.Student() != student {
			return FinalGrade{}, fmt.Errorf("%w (студент №%d, критерий №%d)",
				ErrForeignEvaluation, e.Student(), e.Criterion())
		}
		if _, err := rubric.Criterion(e.Criterion()); err != nil {
			return FinalGrade{}, err
		}
		if _, dup := byCriterion[e.Criterion()]; dup {
			return FinalGrade{}, fmt.Errorf("%w (критерий №%d)", ErrDuplicateEvaluation, e.Criterion())
		}
		byCriterion[e.Criterion()] = e
	}

	criteria := rubric.Criteria()
	lines := make([]FinalLine, len(criteria))
	for i, c := range criteria {
		lines[i].Criterion = c
		if e, ok := byCriterion[c.ID()]; ok {
			lines[i].Evaluation = &e
		}
	}

	value := s.policy.Compute(slices.Clone(lines))
	if !isFinite(value) || value < 0 || value > MaxFinal {
		return FinalGrade{}, fmt.Errorf("правило %q вернуло %s: %w",
			s.policy.Name(), formatNumber(value), ErrFinalOutOfRange)
	}
	return FinalGrade{student: student, value: value, lines: lines, policy: s.policy.Name()}, nil
}
