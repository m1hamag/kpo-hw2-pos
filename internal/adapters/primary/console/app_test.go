package console_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pos/internal/adapters/primary/console"
	"pos/internal/adapters/secondary/csvexport"
	"pos/internal/adapters/secondary/jsonstore"
	"pos/internal/app"
	"pos/internal/domain"
)

var today = time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC)

// run собирает приложение на хранилище в памяти и прогоняет ввод in
func run(t *testing.T, in io.Reader, out io.Writer) error {
	t.Helper()
	store, err := jsonstore.Open("")
	if err != nil {
		t.Fatal(err)
	}
	gb := app.New(store.Students(), store.Rubric(), store.Evaluations(), csvexport.New(),
		[]domain.FinalPolicy{domain.WeightedPolicy{}, domain.GradedOnlyPolicy{}},
		func() time.Time { return today })
	return console.New(gb, in, out).Run()
}

// runLines прогоняет строки ввода и возвращает весь вывод
func runLines(t *testing.T, lines ...string) string {
	t.Helper()
	var out bytes.Buffer
	if err := run(t, strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func assertContains(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(out, p) {
			t.Errorf("в выводе нет %q", p)
		}
	}
	if t.Failed() {
		t.Logf("вывод:\n%s", out)
	}
}

func TestDemo_WholeScenario(t *testing.T) {
	var out bytes.Buffer
	if err := run(t, console.DemoInput(&out), &out); err != nil {
		t.Fatal(err)
	}
	assertContains(t, out.String(),
		// рубрикатор и защита правил при его заполнении
		`Добавлен критерий №1 "Домашние задания", максимум 10, вес 0.4`,
		`Критерий не добавлен: такой критерий уже есть в рубрикаторе: "Домашние задания"`,
		"Критерий не добавлен: максимум критерия должен быть больше 0",
		"№2   Контрольная          максимум 5    вес 0.2    20% итога",
		// студенты
		"Добавлен студент №3 Сидоров Пётр",
		// оценки
		`Оценка не выставлена: критерий "Домашние задания": балл выходит за пределы критерия: 12, можно от 0 до 10`,
		`Иванов Иван получает 8 из 10 по критерию "Домашние задания"`,
		"Оценка не выставлена: оценка по этому критерию уже стоит, её можно исправить",
		"  ! такого номера нет в списке",
		"Оценка исправлена, было 2 из 5, стало 4 из 5",
		"Домашние задания (из 10)   8     16.11.2026  хорошо",
		"Контрольная (из 5)         нет оценки",
		"Итог 6.60, оценка 7, выставлено 2 из 3",
		// итоги при двух правилах
		"№1   Иванов Иван          итог  7.60   оценка  8   выставлено 3 из 3",
		"Теперь итог считается только по выставленным оценкам",
		"№2   Петрова Анна         итог  8.25   оценка  8   выставлено 2 из 3",
		// удаление
		"Критерий не удалён: по критерию уже стоят оценки, сначала отмените их (оценок 2)",
		`Оценка 4 из 5 по критерию "Контрольная" отменена`,
		"Студент №3 Сидоров Пётр удалён",
		// выгрузка
		"№;Студент;Домашние задания (из 10);Контрольная (из 5);Экзамен (из 10);Итог;Оценка;Правило\n"+
			"1;Иванов Иван;8;5;6;7,60;8;по всем критериям, без оценки 0 баллов\n"+
			"2;Петрова Анна;9,5;;7;6,60;7;по всем критериям, без оценки 0 баллов\n",
		"Выгружено студентов 2",
		"До встречи!",
	)
}

func TestEmptyGradebook(t *testing.T) {
	out := runLines(t,
		"2",  // студентов нет
		"5",  // критериев нет
		"7",  // оценку ставить некому
		"11", // итогов нет
		"1", "Иванов Иван",
		"7", "1", // критериев нет
		"8", "1", // без рубрикатора оценок не получить
		"10", "1", // итог без рубрикатора
		"11",
		"0",
	)
	assertContains(t, out,
		"Студентов пока нет\n",
		"Критериев пока нет\n",
		"Студентов пока нет, сначала добавьте студента",
		"Критериев пока нет, сначала добавьте критерий",
		"Не удалось получить оценки: в рубрикаторе нет ни одного критерия",
		"Итог не посчитан: в рубрикаторе нет ни одного критерия",
		"Итоги не посчитаны: в рубрикаторе нет ни одного критерия",
	)
}

func TestInputIsAskedAgain(t *testing.T) {
	out := runLines(t,
		"42", "меню", "", // неверный пункт, не число, показать меню
		"1", "", "Иванов Иван", // пустое ФИО
		"4", "ДЗ", "много", "10", "1,5", // не число
		"3", "1", "может быть", "н", // ни да ни нет, потом отказ
		"12", "3", "2",
		"0",
	)
	assertContains(t, out,
		"  ! нужно целое число от 0 до 13",
		"  ! значение не может быть пустым",
		"  ! нужно число, дробную часть можно писать через точку или запятую",
		`Добавлен критерий №1 "ДЗ", максимум 10, вес 1.5`,
		"  ! ответьте д или н",
		"Удаление отменено",
		"  ! нужно целое число от 1 до 2",
	)
	if strings.Count(out, "Студенты и рубрикатор\n") != 2 {
		t.Error("по Enter меню должно показаться ещё раз")
	}
	if !strings.Contains(runLines(t, "2", "0"), "Список студентов") {
		t.Error("пункт 2 не открылся")
	}
}

func TestRemoveCriterionAndStudent(t *testing.T) {
	out := runLines(t,
		"4", "ДЗ", "10", "1",
		"6", "1",
		"5",
		"1", "Иванов Иван",
		"3", "1", "да",
		"2",
		"0",
	)
	assertContains(t, out,
		`Критерий №1 "ДЗ" удалён`,
		"Критериев пока нет",
		"Студент №1 Иванов Иван удалён",
		"Студентов пока нет",
	)
}

func TestCorrect_DomainRejectsNewScore(t *testing.T) {
	out := runLines(t,
		"4", "ДЗ", "10", "1",
		"1", "Иванов Иван",
		"7", "1", "1", "7", "",
		"8", "1", "1", "11", "",
		"10", "1",
		"0",
	)
	assertContains(t, out,
		`Оценка не исправлена: критерий "ДЗ": балл выходит за пределы критерия: 11, можно от 0 до 10`,
		"ДЗ (из 10)                 7     16.11.2026",
	)
}

func TestExportToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "итоги.csv")
	out := runLines(t,
		"4", "ДЗ", "10", "1",
		"1", "Иванов Иван",
		"7", "1", "1", "8", "",
		"13", path,
		"13", filepath.Join(t.TempDir(), "нет папки", "итоги.csv"),
		"0",
	)
	assertContains(t, out, "Выгружено студентов 1 в файл "+path, "Итоги не выгружены: ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "№;Студент;ДЗ (из 10);Итог;Оценка;Правило\n1;Иванов Иван;8;8,00;8;по всем критериям, без оценки 0 баллов\n"
	if string(data) != want {
		t.Errorf("файл:\n%s\nожидали:\n%s", data, want)
	}
}

func TestEndOfInput(t *testing.T) {
	var out bytes.Buffer
	if err := run(t, strings.NewReader("1\nИванов"), &out); err != nil {
		t.Fatal(err)
	}
	assertContains(t, out.String(), "Добавлен студент №1 Иванов", "Ввод закончился, работа завершена.")
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) {
	return 0, errors.New("клавиатура сломалась")
}

func TestReadError(t *testing.T) {
	err := run(t, brokenReader{}, io.Discard)
	if err == nil || err.Error() != "клавиатура сломалась" {
		t.Errorf("ошибка чтения потерялась: %v", err)
	}
}

func TestDemoInput_WriteError(t *testing.T) {
	in := console.DemoInput(failingWriter{})
	if _, err := in.Read(make([]byte, 10)); err == nil {
		t.Error("ошибка вывода эха потерялась")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("экран погас") }
