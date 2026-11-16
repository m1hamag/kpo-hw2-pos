package console

import (
	"fmt"
	"os"

	"pos/internal/domain"
)

const dateLayout = "02.01.2006"

func (a *App) addStudent() error {
	name, err := a.askText("ФИО")
	if err != nil {
		return err
	}
	s, err := a.gb.AddStudent(name)
	if err != nil {
		a.fail("Студент не добавлен", err)
		return nil
	}
	a.printf("Добавлен студент №%d %s\n", s.ID(), s.Name())
	return nil
}

func (a *App) listStudents() error {
	students, err := a.gb.Students()
	if err != nil {
		a.fail("Не удалось получить студентов", err)
		return nil
	}
	if len(students) == 0 {
		a.printf("Студентов пока нет\n")
		return nil
	}
	for _, s := range students {
		a.printf("  №%-3d %s\n", s.ID(), s.Name())
	}
	return nil
}

// pickStudent показывает студентов и спрашивает номер нужного
func (a *App) pickStudent() (domain.Student, error) {
	students, err := a.gb.Students()
	if err != nil {
		a.fail("Не удалось получить студентов", err)
		return domain.Student{}, errNothing
	}
	if len(students) == 0 {
		a.printf("Студентов пока нет, сначала добавьте студента\n")
		return domain.Student{}, errNothing
	}
	known := make(map[domain.StudentID]domain.Student, len(students))
	for _, s := range students {
		known[s.ID()] = s
		a.printf("  №%-3d %s\n", s.ID(), s.Name())
	}
	return askID(a, "Номер студента", known)
}

func (a *App) removeStudent() error {
	s, err := a.pickStudent()
	if err != nil {
		return quiet(err)
	}
	ok, err := a.askYesNo(fmt.Sprintf("Удалить студента №%d %s вместе со всеми оценками?", s.ID(), s.Name()))
	if err != nil {
		return err
	}
	if !ok {
		a.printf("Удаление отменено\n")
		return nil
	}
	if err := a.gb.RemoveStudent(s.ID()); err != nil {
		a.fail("Студент не удалён", err)
		return nil
	}
	a.printf("Студент №%d %s удалён\n", s.ID(), s.Name())
	return nil
}

func (a *App) addCriterion() error {
	name, err := a.askText("Название критерия")
	if err != nil {
		return err
	}
	maxScore, err := a.askNumber("Максимальный балл")
	if err != nil {
		return err
	}
	weight, err := a.askNumber("Вес в итоге")
	if err != nil {
		return err
	}
	c, err := a.gb.AddCriterion(name, maxScore, weight)
	if err != nil {
		a.fail("Критерий не добавлен", err)
		return nil
	}
	a.printf("Добавлен критерий №%d %q, максимум %s, вес %s\n",
		c.ID(), c.Name(), number(c.MaxScore()), number(c.Weight()))
	return nil
}

func (a *App) showRubric() error {
	criteria, err := a.gb.Criteria()
	if err != nil {
		a.fail("Не удалось получить рубрикатор", err)
		return nil
	}
	if len(criteria) == 0 {
		a.printf("Критериев пока нет\n")
		return nil
	}
	var total float64
	for _, c := range criteria {
		total += c.Weight()
	}
	for _, c := range criteria {
		a.printf("  №%-3d %-20s максимум %-4s вес %-5s %3.0f%% итога\n",
			c.ID(), c.Name(), number(c.MaxScore()), number(c.Weight()), 100*c.Weight()/total)
	}
	return nil
}

// pickCriterion показывает рубрикатор и спрашивает номер критерия
func (a *App) pickCriterion() (domain.Criterion, error) {
	criteria, err := a.gb.Criteria()
	if err != nil {
		a.fail("Не удалось получить рубрикатор", err)
		return domain.Criterion{}, errNothing
	}
	if len(criteria) == 0 {
		a.printf("Критериев пока нет, сначала добавьте критерий\n")
		return domain.Criterion{}, errNothing
	}
	known := make(map[domain.CriterionID]domain.Criterion, len(criteria))
	for _, c := range criteria {
		known[c.ID()] = c
		a.printf("  №%-3d %s (из %s)\n", c.ID(), c.Name(), number(c.MaxScore()))
	}
	return askID(a, "Номер критерия", known)
}

func (a *App) removeCriterion() error {
	c, err := a.pickCriterion()
	if err != nil {
		return quiet(err)
	}
	if err := a.gb.RemoveCriterion(c.ID()); err != nil {
		a.fail("Критерий не удалён", err)
		return nil
	}
	a.printf("Критерий №%d %q удалён\n", c.ID(), c.Name())
	return nil
}

func (a *App) grade() error {
	s, err := a.pickStudent()
	if err != nil {
		return quiet(err)
	}
	c, err := a.pickCriterion()
	if err != nil {
		return quiet(err)
	}
	points, err := a.askNumber(fmt.Sprintf("Балл (максимум %s)", number(c.MaxScore())))
	if err != nil {
		return err
	}
	comment, err := a.askOptional("Комментарий (Enter, чтобы пропустить)")
	if err != nil {
		return err
	}
	e, err := a.gb.Grade(s.ID(), c.ID(), points, comment)
	if err != nil {
		a.fail("Оценка не выставлена", err)
		return nil
	}
	a.printf("%s получает %s по критерию %q\n", s.Name(), e.Score(), c.Name())
	return nil
}

// pickEvaluation показывает выставленные оценки студента и спрашивает, какую взять
func (a *App) pickEvaluation(s domain.Student) (domain.FinalLine, error) {
	f, err := a.gb.Final(s.ID())
	if err != nil {
		a.fail("Не удалось получить оценки", err)
		return domain.FinalLine{}, errNothing
	}
	known := make(map[domain.CriterionID]domain.FinalLine)
	for _, l := range f.Final.Lines() {
		if l.Evaluation == nil {
			continue
		}
		known[l.Criterion.ID()] = l
		a.printf("  №%-3d %-20s %s\n", l.Criterion.ID(), l.Criterion.Name(), l.Evaluation.Score())
	}
	if len(known) == 0 {
		a.printf("У студента пока нет оценок\n")
		return domain.FinalLine{}, errNothing
	}
	return askID(a, "Номер критерия", known)
}

func (a *App) correct() error {
	s, err := a.pickStudent()
	if err != nil {
		return quiet(err)
	}
	l, err := a.pickEvaluation(s)
	if err != nil {
		return quiet(err)
	}
	points, err := a.askNumber(fmt.Sprintf("Новый балл (максимум %s)", number(l.Criterion.MaxScore())))
	if err != nil {
		return err
	}
	comment, err := a.askOptional("Комментарий (Enter, чтобы пропустить)")
	if err != nil {
		return err
	}
	e, err := a.gb.Correct(s.ID(), l.Criterion.ID(), points, comment)
	if err != nil {
		a.fail("Оценка не исправлена", err)
		return nil
	}
	a.printf("Оценка исправлена, было %s, стало %s\n", l.Evaluation.Score(), e.Score())
	return nil
}

func (a *App) cancel() error {
	s, err := a.pickStudent()
	if err != nil {
		return quiet(err)
	}
	l, err := a.pickEvaluation(s)
	if err != nil {
		return quiet(err)
	}
	if err := a.gb.Cancel(s.ID(), l.Criterion.ID()); err != nil {
		a.fail("Оценка не отменена", err)
		return nil
	}
	a.printf("Оценка %s по критерию %q отменена\n", l.Evaluation.Score(), l.Criterion.Name())
	return nil
}

func (a *App) showStudent() error {
	s, err := a.pickStudent()
	if err != nil {
		return quiet(err)
	}
	f, err := a.gb.Final(s.ID())
	if err != nil {
		a.fail("Итог не посчитан", err)
		return nil
	}
	a.printf("\n%s\n", s.Name())
	for _, l := range f.Final.Lines() {
		title := fmt.Sprintf("%s (из %s)", l.Criterion.Name(), number(l.Criterion.MaxScore()))
		if l.Evaluation == nil {
			a.printf("  %-26s нет оценки\n", title)
			continue
		}
		e := l.Evaluation
		line := fmt.Sprintf("  %-26s %-5s %s", title, number(e.Score().Points()), e.Date().Format(dateLayout))
		if e.Comment() != "" {
			line += "  " + e.Comment()
		}
		a.printf("%s\n", line)
	}
	g := f.Final
	a.printf("Итог %.2f, оценка %d, выставлено %d из %d\n", g.Value(), g.Rounded(), g.Graded(), len(g.Lines()))
	a.printf("Итог считается %s\n", g.Policy())
	return nil
}

func (a *App) showFinals() error {
	finals, err := a.gb.Finals()
	if err != nil {
		a.fail("Итоги не посчитаны", err)
		return nil
	}
	if len(finals) == 0 {
		a.printf("Студентов пока нет\n")
		return nil
	}
	a.printf("Итог считается %s\n", a.gb.Policy().Name())
	for _, f := range finals {
		g := f.Final
		a.printf("  №%-3d %-20s итог %5.2f   оценка %2d   выставлено %d из %d\n",
			f.Student.ID(), f.Student.Name(), g.Value(), g.Rounded(), g.Graded(), len(g.Lines()))
	}
	return nil
}

func (a *App) choosePolicy() error {
	policies := a.gb.Policies()
	current := a.gb.Policy().Name()
	for i, p := range policies {
		mark := ""
		if p.Name() == current {
			mark = "   (сейчас)"
		}
		a.printf("  %d. %s%s\n", i+1, p.Name(), mark)
	}
	n, err := a.askInt("Правило", 1, len(policies))
	if err != nil {
		return err
	}
	if err := a.gb.UsePolicy(n - 1); err != nil {
		a.fail("Правило не поменялось", err)
		return nil
	}
	a.printf("Теперь итог считается %s\n", a.gb.Policy().Name())
	return nil
}

func (a *App) export() error {
	path, err := a.askOptional("Файл для выгрузки (Enter, чтобы вывести на экран)")
	if err != nil {
		return err
	}
	if path == "" {
		n, err := a.gb.ExportFinals(a.out)
		if err != nil {
			a.fail("Итоги не выгружены", err)
			return nil
		}
		a.printf("Выгружено студентов %d\n", n)
		return nil
	}

	f, err := os.Create(path)
	if err != nil {
		a.fail("Итоги не выгружены", err)
		return nil
	}
	n, err := a.gb.ExportFinals(f)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		a.fail("Итоги не выгружены", err)
		return nil
	}
	a.printf("Выгружено студентов %d в файл %s\n", n, path)
	return nil
}
