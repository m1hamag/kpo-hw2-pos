package jsonstore_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pos/internal/adapters/secondary/jsonstore"
	"pos/internal/domain"
)

var day = time.Date(2026, time.November, 16, 0, 0, 0, 0, time.UTC)

// fill заводит двух студентов, два критерия и три оценки
func fill(t *testing.T, s *jsonstore.Store) {
	t.Helper()
	for i, name := range []string{"Иванов Иван", "Петрова Анна"} {
		st, _ := domain.NewStudent(domain.StudentID(i+1), name)
		if err := s.Students().Save(st); err != nil {
			t.Fatal(err)
		}
	}
	r, _ := s.Rubric().Load()
	r.AddCriterion("ДЗ", 10, 0.4)
	r.AddCriterion("Экзамен", 10, 0.6)
	if err := s.Rubric().Save(r); err != nil {
		t.Fatal(err)
	}
	f := domain.NewEvaluationFactory(r)
	for _, e := range []struct {
		student   domain.StudentID
		criterion domain.CriterionID
		points    float64
		comment   string
	}{{1, 1, 8, "хорошо"}, {1, 2, 6.5, ""}, {2, 1, 10, ""}} {
		ev, err := f.Create(e.student, e.criterion, e.points, day, e.comment)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Evaluations().Save(ev); err != nil {
			t.Fatal(err)
		}
	}
}

func memory(t *testing.T) *jsonstore.Store {
	t.Helper()
	s, err := jsonstore.Open("")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStudents(t *testing.T) {
	repo := memory(t).Students()
	if id, _ := repo.NextID(); id != 1 {
		t.Errorf("первый номер %d", id)
	}
	b, _ := domain.NewStudent(2, "Петрова Анна")
	a, _ := domain.NewStudent(1, "Иванов Иван")
	repo.Save(b)
	repo.Save(a)
	if id, _ := repo.NextID(); id != 3 {
		t.Errorf("следующий номер %d, ожидали 3", id)
	}

	list, _ := repo.List()
	if len(list) != 2 || list[0] != a || list[1] != b {
		t.Errorf("список %v, ожидали по возрастанию номера", list)
	}
	if got, err := repo.Get(2); err != nil || got != b {
		t.Errorf("Get(2) = %v, %v", got, err)
	}

	renamed, _ := domain.NewStudent(2, "Петрова-Смирнова Анна")
	repo.Save(renamed)
	if got, _ := repo.Get(2); got.Name() != "Петрова-Смирнова Анна" {
		t.Errorf("Save не заменил студента, получили %q", got.Name())
	}

	if err := repo.Delete(1); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(1); !errors.Is(err, domain.ErrStudentNotFound) {
		t.Errorf("Get удалённого: %v", err)
	}
	if err := repo.Delete(1); !errors.Is(err, domain.ErrStudentNotFound) {
		t.Errorf("повторный Delete: %v", err)
	}
}

func TestRubric_LoadReturnsCopy(t *testing.T) {
	repo := memory(t).Rubric()
	r, _ := repo.Load()
	if !r.IsEmpty() {
		t.Fatal("в новом хранилище рубрикатор должен быть пустым")
	}
	r.AddCriterion("ДЗ", 10, 1)
	if again, _ := repo.Load(); !again.IsEmpty() {
		t.Error("рубрикатор поменялся без Save")
	}
	repo.Save(r)
	if again, _ := repo.Load(); len(again.Criteria()) != 1 {
		t.Error("после Save критерий не сохранился")
	}
	if err := repo.Save(nil); err == nil {
		t.Error("Save(nil) должен вернуть ошибку")
	}
}

func TestEvaluations(t *testing.T) {
	s := memory(t)
	fill(t, s)
	repo := s.Evaluations()

	byStudent, _ := repo.ByStudent(1)
	if len(byStudent) != 2 || byStudent[0].Criterion() != 1 || byStudent[1].Criterion() != 2 {
		t.Errorf("ByStudent(1) %v", byStudent)
	}
	byCriterion, _ := repo.ByCriterion(1)
	if len(byCriterion) != 2 || byCriterion[0].Student() != 1 || byCriterion[1].Student() != 2 {
		t.Errorf("ByCriterion(1) %v", byCriterion)
	}

	key := domain.EvaluationKey{Student: 1, Criterion: 1}
	r, _ := s.Rubric().Load()
	fixed, _ := domain.NewEvaluationFactory(r).Create(1, 1, 9, day, "пересдача")
	repo.Save(fixed)
	if got, _ := repo.Get(key); got.Score().Points() != 9 || got.Comment() != "пересдача" {
		t.Errorf("оценка не заменилась, получили %v", got.Score())
	}
	if all, _ := repo.ByStudent(1); len(all) != 2 {
		t.Errorf("после исправления оценок %d, ожидали 2", len(all))
	}

	if err := repo.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(key); !errors.Is(err, domain.ErrEvaluationNotFound) {
		t.Errorf("Get удалённой: %v", err)
	}
	if err := repo.Delete(key); !errors.Is(err, domain.ErrEvaluationNotFound) {
		t.Errorf("повторный Delete: %v", err)
	}

	if err := repo.DeleteByStudent(1); err != nil {
		t.Fatal(err)
	}
	if left, _ := repo.ByStudent(1); len(left) != 0 {
		t.Errorf("у студента 1 остались оценки %v", left)
	}
	if other, _ := repo.ByStudent(2); len(other) != 1 {
		t.Error("DeleteByStudent задел другого студента")
	}
	if err := repo.DeleteByStudent(1); err != nil {
		t.Errorf("удаление без оценок: %v", err)
	}
}

func TestStore_RefusesStateThatWouldNotLoad(t *testing.T) {
	s := memory(t)
	fill(t, s)

	// оценка студенту, которого нет
	r, _ := s.Rubric().Load()
	ghost, _ := domain.NewEvaluationFactory(r).Create(99, 1, 5, day, "")
	if err := s.Evaluations().Save(ghost); !errors.Is(err, domain.ErrStudentNotFound) {
		t.Errorf("оценка без студента: %v", err)
	}
	if _, err := s.Evaluations().Get(ghost.Key()); err == nil {
		t.Error("оценка без студента осталась в хранилище")
	}

	// критерий, по которому стоят оценки
	r.RemoveCriterion(1)
	if err := s.Rubric().Save(r); !errors.Is(err, domain.ErrUnknownCriterion) {
		t.Errorf("удаление критерия с оценками: %v", err)
	}
	if again, _ := s.Rubric().Load(); len(again.Criteria()) != 2 {
		t.Error("рубрикатор поменялся, хотя сохранение отклонено")
	}

	// студент, у которого есть оценки
	if err := s.Students().Delete(1); !errors.Is(err, domain.ErrStudentNotFound) {
		t.Errorf("удаление студента с оценками: %v", err)
	}
	if _, err := s.Students().Get(1); err != nil {
		t.Error("студент удалился, хотя у него есть оценки")
	}
}

func TestFile_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pos.json")
	s, err := jsonstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	fill(t, s)

	loaded, err := jsonstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	students, _ := loaded.Students().List()
	if len(students) != 2 || students[1].Name() != "Петрова Анна" {
		t.Errorf("студенты %v", students)
	}
	r, _ := loaded.Rubric().Load()
	if c, err := r.Criterion(2); err != nil || c.Name() != "Экзамен" || c.Weight() != 0.6 {
		t.Errorf("критерий 2 %+v, %v", c, err)
	}
	e, err := loaded.Evaluations().Get(domain.EvaluationKey{Student: 1, Criterion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if e.Score().String() != "8 из 10" || !e.Date().Equal(day) || e.Comment() != "хорошо" {
		t.Errorf("оценка %v, %v, %q", e.Score(), e.Date(), e.Comment())
	}
}

func TestFile_Format(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pos.json")
	s, _ := jsonstore.Open(path)
	fill(t, s)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "students": [
    {
      "id": 1,
      "name": "Иванов Иван"
    },
    {
      "id": 2,
      "name": "Петрова Анна"
    }
  ],
  "criteria": [
    {
      "id": 1,
      "name": "ДЗ",
      "max_score": 10,
      "weight": 0.4
    },
    {
      "id": 2,
      "name": "Экзамен",
      "max_score": 10,
      "weight": 0.6
    }
  ],
  "evaluations": [
    {
      "student_id": 1,
      "criterion_id": 1,
      "score": 8,
      "date": "2026-11-16",
      "comment": "хорошо"
    },
    {
      "student_id": 1,
      "criterion_id": 2,
      "score": 6.5,
      "date": "2026-11-16"
    },
    {
      "student_id": 2,
      "criterion_id": 1,
      "score": 10,
      "date": "2026-11-16"
    }
  ]
}
`
	if string(data) != want {
		t.Errorf("файл:\n%s\nожидали:\n%s", data, want)
	}
}

func TestOpen_RejectsBrokenFile(t *testing.T) {
	const students = `"students": [{"id": 1, "name": "Иванов Иван"}]`
	const criteria = `"criteria": [{"id": 1, "name": "ДЗ", "max_score": 10, "weight": 1}]`
	tests := []struct {
		name string
		json string
		want error
	}{
		{"12 из 10", `{` + students + `,` + criteria + `, "evaluations": [{"student_id": 1, "criterion_id": 1, "score": 12, "date": "2026-11-16"}]}`, domain.ErrScoreOutOfRange},
		{"критерия нет", `{` + students + `,` + criteria + `, "evaluations": [{"student_id": 1, "criterion_id": 7, "score": 5, "date": "2026-11-16"}]}`, domain.ErrUnknownCriterion},
		{"студента нет", `{` + students + `,` + criteria + `, "evaluations": [{"student_id": 5, "criterion_id": 1, "score": 5, "date": "2026-11-16"}]}`, domain.ErrStudentNotFound},
		{"две оценки по критерию", `{` + students + `,` + criteria + `, "evaluations": [{"student_id": 1, "criterion_id": 1, "score": 5, "date": "2026-11-16"}, {"student_id": 1, "criterion_id": 1, "score": 6, "date": "2026-11-16"}]}`, domain.ErrDuplicateEvaluation},
		{"пустое ФИО", `{"students": [{"id": 1, "name": " "}]}`, domain.ErrEmptyName},
		{"повтор критерия", `{"criteria": [{"id": 1, "name": "ДЗ", "max_score": 10, "weight": 1}, {"id": 2, "name": "дз", "max_score": 5, "weight": 1}]}`, domain.ErrDuplicateCriterion},
		{"вес 0", `{"criteria": [{"id": 1, "name": "ДЗ", "max_score": 10, "weight": 0}]}`, domain.ErrInvalidWeight},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pos.json")
			os.WriteFile(path, []byte(tt.json), 0o644)
			if _, err := jsonstore.Open(path); !errors.Is(err, tt.want) {
				t.Errorf("ожидали %v, получили %v", tt.want, err)
			}
		})
	}

	for name, content := range map[string]string{
		"не JSON":         `{"students": [`,
		"повтор студента": `{"students": [{"id": 1, "name": "А"}, {"id": 1, "name": "Б"}]}`,
		"неверная дата":   `{` + students + `,` + criteria + `, "evaluations": [{"student_id": 1, "criterion_id": 1, "score": 5, "date": "16.11.2026"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pos.json")
			os.WriteFile(path, []byte(content), 0o644)
			_, err := jsonstore.Open(path)
			if err == nil || !strings.Contains(err.Error(), path) {
				t.Errorf("ожидали ошибку с именем файла, получили %v", err)
			}
		})
	}
}

func TestOpen_ReadError(t *testing.T) {
	if _, err := jsonstore.Open(t.TempDir()); err == nil {
		t.Error("папку вместо файла прочитать нельзя")
	}
}

func TestFile_WriteErrorRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "нет такой папки", "pos.json")
	s, err := jsonstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := domain.NewStudent(1, "Иванов Иван")
	if err := s.Students().Save(st); err == nil {
		t.Fatal("ожидали ошибку записи")
	}
	if list, _ := s.Students().List(); len(list) != 0 {
		t.Errorf("после ошибки записи в памяти остался студент %v", list)
	}
}
