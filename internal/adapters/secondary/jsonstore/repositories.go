package jsonstore

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"pos/internal/domain"
)

var (
	_ domain.StudentRepository    = studentRepo{}
	_ domain.RubricRepository     = rubricRepo{}
	_ domain.EvaluationRepository = evaluationRepo{}
)

type studentRepo struct{ s *Store }

func (r studentRepo) NextID() (domain.StudentID, error) {
	var maxID domain.StudentID
	for id := range r.s.students {
		maxID = max(maxID, id)
	}
	return maxID + 1, nil
}

func (r studentRepo) Save(st domain.Student) error {
	old, had := r.s.students[st.ID()]
	r.s.students[st.ID()] = st
	return r.s.commit(func() {
		if had {
			r.s.students[st.ID()] = old
		} else {
			delete(r.s.students, st.ID())
		}
	})
}

func (r studentRepo) Get(id domain.StudentID) (domain.Student, error) {
	st, ok := r.s.students[id]
	if !ok {
		return domain.Student{}, fmt.Errorf("%w (№%d)", domain.ErrStudentNotFound, id)
	}
	return st, nil
}

func (r studentRepo) List() ([]domain.Student, error) {
	return sortedStudents(r.s.students), nil
}

func (r studentRepo) Delete(id domain.StudentID) error {
	old, ok := r.s.students[id]
	if !ok {
		return fmt.Errorf("%w (№%d)", domain.ErrStudentNotFound, id)
	}
	delete(r.s.students, id)
	return r.s.commit(func() { r.s.students[id] = old })
}

type rubricRepo struct{ s *Store }

// Load каждый раз собирает новый рубрикатор, поэтому изменения
// попадут в хранилище только после Save
func (r rubricRepo) Load() (*domain.Rubric, error) {
	return domain.NewRubric(r.s.criteria...)
}

func (r rubricRepo) Save(rubric *domain.Rubric) error {
	if rubric == nil {
		return errors.New("нечего сохранять, рубрикатор не передан")
	}
	old := r.s.criteria
	r.s.criteria = rubric.Criteria()
	return r.s.commit(func() { r.s.criteria = old })
}

type evaluationRepo struct{ s *Store }

func (r evaluationRepo) Save(e domain.Evaluation) error {
	old, had := r.s.evaluations[e.Key()]
	r.s.evaluations[e.Key()] = e
	return r.s.commit(func() {
		if had {
			r.s.evaluations[e.Key()] = old
		} else {
			delete(r.s.evaluations, e.Key())
		}
	})
}

func (r evaluationRepo) Get(key domain.EvaluationKey) (domain.Evaluation, error) {
	e, ok := r.s.evaluations[key]
	if !ok {
		return domain.Evaluation{}, notFound(key)
	}
	return e, nil
}

func (r evaluationRepo) ByStudent(id domain.StudentID) ([]domain.Evaluation, error) {
	list := r.filter(func(e domain.Evaluation) bool { return e.Student() == id })
	slices.SortFunc(list, func(a, b domain.Evaluation) int { return cmp.Compare(a.Criterion(), b.Criterion()) })
	return list, nil
}

func (r evaluationRepo) ByCriterion(id domain.CriterionID) ([]domain.Evaluation, error) {
	list := r.filter(func(e domain.Evaluation) bool { return e.Criterion() == id })
	slices.SortFunc(list, func(a, b domain.Evaluation) int { return cmp.Compare(a.Student(), b.Student()) })
	return list, nil
}

func (r evaluationRepo) Delete(key domain.EvaluationKey) error {
	old, ok := r.s.evaluations[key]
	if !ok {
		return notFound(key)
	}
	delete(r.s.evaluations, key)
	return r.s.commit(func() { r.s.evaluations[key] = old })
}

func (r evaluationRepo) DeleteByStudent(id domain.StudentID) error {
	removed, _ := r.ByStudent(id)
	if len(removed) == 0 {
		return nil
	}
	for _, e := range removed {
		delete(r.s.evaluations, e.Key())
	}
	return r.s.commit(func() {
		for _, e := range removed {
			r.s.evaluations[e.Key()] = e
		}
	})
}

func (r evaluationRepo) filter(keep func(domain.Evaluation) bool) []domain.Evaluation {
	var list []domain.Evaluation
	for _, e := range r.s.evaluations {
		if keep(e) {
			list = append(list, e)
		}
	}
	return list
}

func notFound(key domain.EvaluationKey) error {
	return fmt.Errorf("%w (студент №%d, критерий №%d)", domain.ErrEvaluationNotFound, key.Student, key.Criterion)
}
