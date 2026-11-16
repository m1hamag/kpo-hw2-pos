// Package console это консольный интерфейс ПОС, primary-адаптер.
// Он только спрашивает и печатает, а все сценарии вызывает через фасад app.Gradebook
package console

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"

	"pos/internal/app"
)

const banner = `==================================================
   ПОС. Преподавательская оценка студентов
==================================================`

// App это консольное меню модуля ПОС
type App struct {
	gb  *app.Gradebook
	in  *bufio.Scanner
	out io.Writer
}

// New создаёт меню, которое читает команды из in и пишет в out
func New(gb *app.Gradebook, in io.Reader, out io.Writer) *App {
	return &App{gb: gb, in: bufio.NewScanner(in), out: out}
}

type menuItem struct {
	title  string
	action func() error
}

type menuSection struct {
	title string
	items []menuItem
}

// Run показывает меню и выполняет команды до выхода или конца ввода
func (a *App) Run() error {
	a.printf("%s\n", banner)
	err := a.loop()
	if errors.Is(err, io.EOF) {
		a.printf("\nВвод закончился, работа завершена.\n")
		return nil
	}
	return err
}

func (a *App) menu() []menuSection {
	return []menuSection{
		{"Студенты и рубрикатор", []menuItem{
			{"Добавить студента", a.addStudent},
			{"Список студентов", a.listStudents},
			{"Удалить студента", a.removeStudent},
			{"Добавить критерий", a.addCriterion},
			{"Рубрикатор", a.showRubric},
			{"Удалить критерий", a.removeCriterion},
		}},
		{"Оценки", []menuItem{
			{"Выставить оценку", a.grade},
			{"Исправить оценку", a.correct},
			{"Отменить оценку", a.cancel},
			{"Оценки студента", a.showStudent},
		}},
		{"Итог", []menuItem{
			{"Итоги всех студентов", a.showFinals},
			{"Правило подсчёта итога", a.choosePolicy},
			{"Выгрузить итоги в CSV", a.export},
		}},
	}
}

// showMenu это номер пункта, который просит заново показать меню
const showMenu = -1

func (a *App) loop() error {
	sections := a.menu()
	var items []menuItem
	for _, s := range sections {
		items = append(items, s.items...)
	}
	prompt := fmt.Sprintf("Пункт меню (0-%d, Enter покажет меню)", len(items))
	parse := func(s string) (int, error) {
		if s == "" {
			return showMenu, nil
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n > len(items) {
			return 0, fmt.Errorf("нужно целое число от 0 до %d", len(items))
		}
		return n, nil
	}

	a.printMenu(sections)
	for {
		a.printf("\n")
		choice, err := ask(a, prompt, parse)
		if err != nil {
			return err
		}
		switch choice {
		case showMenu:
			a.printMenu(sections)
			continue
		case 0:
			a.printf("До встречи!\n")
			return nil
		}
		item := items[choice-1]
		a.printf("[%s]\n", item.title)
		if err := item.action(); err != nil {
			return err
		}
	}
}

func (a *App) printMenu(sections []menuSection) {
	a.printf("\n")
	n := 0
	for _, s := range sections {
		a.printf("%s\n", s.title)
		for _, item := range s.items {
			n++
			a.printf("  %2d. %s\n", n, item.title)
		}
	}
	a.printf("   0. Выход\n")
}
