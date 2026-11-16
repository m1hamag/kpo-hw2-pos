// Package jsonstore хранит данные ПОС в JSON-файле и реализует порты репозиториев домена.
// Это secondary-адаптер, домен про него ничего не знает
package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"pos/internal/domain"
)

// Store держит данные в памяти и после каждого изменения переписывает файл.
// С пустым путём файла нет вообще, так удобно для демо и тестов
type Store struct {
	path        string
	students    map[domain.StudentID]domain.Student
	criteria    []domain.Criterion
	evaluations map[domain.EvaluationKey]domain.Evaluation
}

func newStore(path string) *Store {
	return &Store{
		path:        path,
		students:    make(map[domain.StudentID]domain.Student),
		evaluations: make(map[domain.EvaluationKey]domain.Evaluation),
	}
}

// Open читает данные из файла path. Если файла ещё нет, хранилище будет пустым
func Open(path string) (*Store, error) {
	s := newStore(path)
	if path == "" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("файл %s: %w", path, err)
	}
	if err := s.restore(snap); err != nil {
		return nil, fmt.Errorf("файл %s: %w", path, err)
	}
	return s, nil
}

// Students возвращает репозиторий студентов
func (s *Store) Students() domain.StudentRepository { return studentRepo{s} }

// Rubric возвращает репозиторий рубрикатора
func (s *Store) Rubric() domain.RubricRepository { return rubricRepo{s} }

// Evaluations возвращает репозиторий оценок
func (s *Store) Evaluations() domain.EvaluationRepository { return evaluationRepo{s} }

// commit проверяет, что новое состояние потом загрузится, и записывает файл.
// Если что-то не так, undo откатывает изменение в памяти
func (s *Store) commit(undo func()) error {
	snap := s.snapshot()
	if err := newStore("").restore(snap); err != nil {
		undo()
		return err
	}
	if s.path == "" {
		return nil
	}
	if err := write(s.path, snap); err != nil {
		undo()
		return err
	}
	return nil
}

// write сначала пишет во временный файл и потом переименовывает его,
// чтобы при сбое старый файл остался целым
func write(path string, snap snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("не удалось сохранить данные: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("не удалось сохранить данные: %w", err)
	}
	return nil
}
