package jsonstore

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"pos/internal/domain"
)

const dateLayout = "2006-01-02"

// snapshot описывает, как данные лежат в файле
type snapshot struct {
	Students    []studentDTO    `json:"students"`
	Criteria    []criterionDTO  `json:"criteria"`
	Evaluations []evaluationDTO `json:"evaluations"`
}

type studentDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type criterionDTO struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	MaxScore float64 `json:"max_score"`
	Weight   float64 `json:"weight"`
}

type evaluationDTO struct {
	Student   int     `json:"student_id"`
	Criterion int     `json:"criterion_id"`
	Score     float64 `json:"score"`
	Date      string  `json:"date"`
	Comment   string  `json:"comment,omitempty"`
}

// snapshot переводит доменные объекты в DTO в стабильном порядке
func (s *Store) snapshot() snapshot {
	snap := snapshot{
		Students:    make([]studentDTO, 0, len(s.students)),
		Criteria:    make([]criterionDTO, 0, len(s.criteria)),
		Evaluations: make([]evaluationDTO, 0, len(s.evaluations)),
	}
	for _, st := range sortedStudents(s.students) {
		snap.Students = append(snap.Students, studentDTO{ID: int(st.ID()), Name: st.Name()})
	}
	for _, c := range s.criteria {
		snap.Criteria = append(snap.Criteria, criterionDTO{
			ID: int(c.ID()), Name: c.Name(), MaxScore: c.MaxScore(), Weight: c.Weight(),
		})
	}
	evals := make([]domain.Evaluation, 0, len(s.evaluations))
	for _, e := range s.evaluations {
		evals = append(evals, e)
	}
	slices.SortFunc(evals, func(a, b domain.Evaluation) int {
		return cmp.Or(cmp.Compare(a.Student(), b.Student()), cmp.Compare(a.Criterion(), b.Criterion()))
	})
	for _, e := range evals {
		snap.Evaluations = append(snap.Evaluations, evaluationDTO{
			Student:   int(e.Student()),
			Criterion: int(e.Criterion()),
			Score:     e.Score().Points(),
			Date:      e.Date().Format(dateLayout),
			Comment:   e.Comment(),
		})
	}
	return snap
}

// restore собирает доменные объекты через их конструкторы и фабрику оценок,
// поэтому из испорченного файла не загрузится ни 12 из 10, ни оценка без критерия
func (s *Store) restore(snap snapshot) error {
	for i, d := range snap.Students {
		st, err := domain.NewStudent(domain.StudentID(d.ID), d.Name)
		if err != nil {
			return fmt.Errorf("студент %d: %w", i+1, err)
		}
		if _, dup := s.students[st.ID()]; dup {
			return fmt.Errorf("студент %d: номер %d уже занят", i+1, st.ID())
		}
		s.students[st.ID()] = st
	}

	criteria := make([]domain.Criterion, 0, len(snap.Criteria))
	for i, d := range snap.Criteria {
		c, err := domain.NewCriterion(domain.CriterionID(d.ID), d.Name, d.MaxScore, d.Weight)
		if err != nil {
			return fmt.Errorf("критерий %d: %w", i+1, err)
		}
		criteria = append(criteria, c)
	}
	rubric, err := domain.NewRubric(criteria...)
	if err != nil {
		return err
	}
	s.criteria = criteria

	factory := domain.NewEvaluationFactory(rubric)
	for i, d := range snap.Evaluations {
		if _, ok := s.students[domain.StudentID(d.Student)]; !ok {
			return fmt.Errorf("оценка %d: %w (№%d)", i+1, domain.ErrStudentNotFound, d.Student)
		}
		date, err := time.Parse(dateLayout, d.Date)
		if err != nil {
			return fmt.Errorf("оценка %d: неверная дата %q", i+1, d.Date)
		}
		e, err := factory.Create(domain.StudentID(d.Student), domain.CriterionID(d.Criterion), d.Score, date, d.Comment)
		if err != nil {
			return fmt.Errorf("оценка %d: %w", i+1, err)
		}
		if _, dup := s.evaluations[e.Key()]; dup {
			return fmt.Errorf("оценка %d: %w", i+1, domain.ErrDuplicateEvaluation)
		}
		s.evaluations[e.Key()] = e
	}
	return nil
}

func sortedStudents(m map[domain.StudentID]domain.Student) []domain.Student {
	list := make([]domain.Student, 0, len(m))
	for _, st := range m {
		list = append(list, st)
	}
	slices.SortFunc(list, func(a, b domain.Student) int { return cmp.Compare(a.ID(), b.ID()) })
	return list
}
