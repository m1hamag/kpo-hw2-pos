package app

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"pos/internal/domain"
)

// Gradebook собирает все сценарии ПОС в одном месте (паттерн "Фасад").
// Консоль не трогает репозитории, фабрику и доменный сервис напрямую
type Gradebook struct {
	students    domain.StudentRepository
	rubrics     domain.RubricRepository
	evaluations domain.EvaluationRepository
	exporter    FinalsExporter
	policies    []domain.FinalPolicy
	finals      domain.FinalGradeService
	now         Clock
}

// New создаёт фасад. Первое правило из policies считается правилом по умолчанию
func New(students domain.StudentRepository, rubrics domain.RubricRepository,
	evaluations domain.EvaluationRepository, exporter FinalsExporter,
	policies []domain.FinalPolicy, now Clock) *Gradebook {
	if len(policies) == 0 {
		policies = []domain.FinalPolicy{domain.WeightedPolicy{}}
	}
	if now == nil {
		now = time.Now
	}
	return &Gradebook{
		students:    students,
		rubrics:     rubrics,
		evaluations: evaluations,
		exporter:    exporter,
		policies:    slices.Clone(policies),
		finals:      domain.NewFinalGradeService(policies[0]),
		now:         now,
	}
}

// AddStudent заводит нового студента
func (g *Gradebook) AddStudent(name string) (domain.Student, error) {
	id, err := g.students.NextID()
	if err != nil {
		return domain.Student{}, err
	}
	s, err := domain.NewStudent(id, name)
	if err != nil {
		return domain.Student{}, err
	}
	if err := g.students.Save(s); err != nil {
		return domain.Student{}, err
	}
	return s, nil
}

// Students возвращает всех студентов
func (g *Gradebook) Students() ([]domain.Student, error) {
	return g.students.List()
}

// RemoveStudent удаляет студента вместе с его оценками
func (g *Gradebook) RemoveStudent(id domain.StudentID) error {
	if _, err := g.students.Get(id); err != nil {
		return err
	}
	if err := g.evaluations.DeleteByStudent(id); err != nil {
		return err
	}
	return g.students.Delete(id)
}

// AddCriterion добавляет критерий в рубрикатор
func (g *Gradebook) AddCriterion(name string, maxScore, weight float64) (domain.Criterion, error) {
	r, err := g.rubrics.Load()
	if err != nil {
		return domain.Criterion{}, err
	}
	c, err := r.AddCriterion(name, maxScore, weight)
	if err != nil {
		return domain.Criterion{}, err
	}
	if err := g.rubrics.Save(r); err != nil {
		return domain.Criterion{}, err
	}
	return c, nil
}

// Criteria возвращает критерии рубрикатора
func (g *Gradebook) Criteria() ([]domain.Criterion, error) {
	r, err := g.rubrics.Load()
	if err != nil {
		return nil, err
	}
	return r.Criteria(), nil
}

// RemoveCriterion убирает критерий, если по нему ещё нет оценок.
// Иначе оценки ссылались бы на критерий, которого нет в рубрикаторе
func (g *Gradebook) RemoveCriterion(id domain.CriterionID) error {
	r, err := g.rubrics.Load()
	if err != nil {
		return err
	}
	if err := r.RemoveCriterion(id); err != nil {
		return err
	}
	graded, err := g.evaluations.ByCriterion(id)
	if err != nil {
		return err
	}
	if len(graded) > 0 {
		return fmt.Errorf("%w (оценок %d)", ErrCriterionInUse, len(graded))
	}
	return g.rubrics.Save(r)
}

// Grade выставляет оценку. Если по этому критерию оценка уже есть, её исправляют через Correct
func (g *Gradebook) Grade(student domain.StudentID, criterion domain.CriterionID,
	points float64, comment string) (domain.Evaluation, error) {
	if _, err := g.students.Get(student); err != nil {
		return domain.Evaluation{}, err
	}
	_, err := g.evaluations.Get(domain.EvaluationKey{Student: student, Criterion: criterion})
	if err == nil {
		return domain.Evaluation{}, ErrAlreadyEvaluated
	}
	if !errors.Is(err, domain.ErrEvaluationNotFound) {
		return domain.Evaluation{}, err
	}
	return g.save(student, criterion, points, comment)
}

// Correct исправляет уже выставленную оценку, дата меняется на сегодняшнюю
func (g *Gradebook) Correct(student domain.StudentID, criterion domain.CriterionID,
	points float64, comment string) (domain.Evaluation, error) {
	if _, err := g.evaluations.Get(domain.EvaluationKey{Student: student, Criterion: criterion}); err != nil {
		return domain.Evaluation{}, err
	}
	return g.save(student, criterion, points, comment)
}

// Cancel отменяет оценку
func (g *Gradebook) Cancel(student domain.StudentID, criterion domain.CriterionID) error {
	return g.evaluations.Delete(domain.EvaluationKey{Student: student, Criterion: criterion})
}

// save создаёт оценку через фабрику по текущему рубрикатору и сохраняет её
func (g *Gradebook) save(student domain.StudentID, criterion domain.CriterionID,
	points float64, comment string) (domain.Evaluation, error) {
	r, err := g.rubrics.Load()
	if err != nil {
		return domain.Evaluation{}, err
	}
	e, err := domain.NewEvaluationFactory(r).Create(student, criterion, points, g.now(), comment)
	if err != nil {
		return domain.Evaluation{}, err
	}
	if err := g.evaluations.Save(e); err != nil {
		return domain.Evaluation{}, err
	}
	return e, nil
}

// Final считает итог студента по текущему правилу
func (g *Gradebook) Final(id domain.StudentID) (StudentFinal, error) {
	s, err := g.students.Get(id)
	if err != nil {
		return StudentFinal{}, err
	}
	r, err := g.rubrics.Load()
	if err != nil {
		return StudentFinal{}, err
	}
	return g.final(s, r)
}

// Finals считает итоги всех студентов
func (g *Gradebook) Finals() ([]StudentFinal, error) {
	students, err := g.students.List()
	if err != nil {
		return nil, err
	}
	r, err := g.rubrics.Load()
	if err != nil {
		return nil, err
	}
	finals := make([]StudentFinal, 0, len(students))
	for _, s := range students {
		f, err := g.final(s, r)
		if err != nil {
			return nil, err
		}
		finals = append(finals, f)
	}
	return finals, nil
}

func (g *Gradebook) final(s domain.Student, r *domain.Rubric) (StudentFinal, error) {
	evals, err := g.evaluations.ByStudent(s.ID())
	if err != nil {
		return StudentFinal{}, err
	}
	f, err := g.finals.Final(s.ID(), r, evals)
	if err != nil {
		return StudentFinal{}, err
	}
	return StudentFinal{Student: s, Final: f}, nil
}

// Policies возвращает доступные правила подсчёта итога
func (g *Gradebook) Policies() []domain.FinalPolicy {
	return slices.Clone(g.policies)
}

// Policy возвращает правило, по которому сейчас считается итог
func (g *Gradebook) Policy() domain.FinalPolicy {
	return g.finals.Policy()
}

// UsePolicy переключает подсчёт итога на правило с номером i из Policies (с нуля)
func (g *Gradebook) UsePolicy(i int) error {
	if i < 0 || i >= len(g.policies) {
		return fmt.Errorf("%w (№%d)", ErrUnknownPolicy, i+1)
	}
	g.finals = domain.NewFinalGradeService(g.policies[i])
	return nil
}

// ExportFinals выгружает итоги всех студентов в w и сообщает, сколько студентов выгружено
func (g *Gradebook) ExportFinals(w io.Writer) (int, error) {
	finals, err := g.Finals()
	if err != nil {
		return 0, err
	}
	if err := g.exporter.Export(w, finals); err != nil {
		return 0, err
	}
	return len(finals), nil
}
