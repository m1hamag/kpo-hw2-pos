package domain

import (
	"fmt"
	"strings"
)

// StudentID это номер студента в ведомости
type StudentID int

// Student описывает студента, которого оценивают
type Student struct {
	id   StudentID
	name string
}

// NewStudent создаёт студента. Номер для него выдаёт StudentRepository.NextID
func NewStudent(id StudentID, name string) (Student, error) {
	if id <= 0 {
		return Student{}, fmt.Errorf("студент: %w (%d)", ErrInvalidID, id)
	}
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return Student{}, fmt.Errorf("ФИО студента %w", ErrEmptyName)
	}
	return Student{id: id, name: name}, nil
}

// ID возвращает номер студента
func (s Student) ID() StudentID { return s.id }

// Name возвращает ФИО студента
func (s Student) Name() string { return s.name }
