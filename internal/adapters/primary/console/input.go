package console

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// errNothing значит, что выбирать не из чего и действие нужно тихо закончить
var errNothing = errors.New("нечего выбирать")

// quiet превращает errNothing в nil, остальные ошибки ввода пропускает дальше
func quiet(err error) error {
	if errors.Is(err, errNothing) {
		return nil
	}
	return err
}

func (a *App) printf(format string, args ...any) {
	fmt.Fprintf(a.out, format, args...)
}

// fail печатает, почему сценарий не выполнился
func (a *App) fail(what string, err error) {
	a.printf("%s: %v\n", what, err)
}

func (a *App) readLine() (string, error) {
	if a.in.Scan() {
		return strings.TrimSpace(a.in.Text()), nil
	}
	if err := a.in.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

// ask спрашивает, пока parse не примет ответ
func ask[T any](a *App, prompt string, parse func(string) (T, error)) (T, error) {
	for {
		a.printf("%s: ", prompt)
		line, err := a.readLine()
		if err != nil {
			var zero T
			return zero, err
		}
		value, err := parse(line)
		if err == nil {
			return value, nil
		}
		a.printf("  ! %v\n", err)
	}
}

func (a *App) askText(prompt string) (string, error) {
	return ask(a, prompt, func(s string) (string, error) {
		if s == "" {
			return "", errors.New("значение не может быть пустым")
		}
		return s, nil
	})
}

// askOptional принимает и пустой ответ
func (a *App) askOptional(prompt string) (string, error) {
	return ask(a, prompt, func(s string) (string, error) { return s, nil })
}

// askNumber проверяет только, что введено число. Подходит ли оно, решает домен
func (a *App) askNumber(prompt string) (float64, error) {
	return ask(a, prompt, func(s string) (float64, error) {
		x, err := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
		if err != nil || math.IsNaN(x) || math.IsInf(x, 0) {
			return 0, errors.New("нужно число, дробную часть можно писать через точку или запятую")
		}
		return x, nil
	})
}

func (a *App) askInt(prompt string, lo, hi int) (int, error) {
	return ask(a, fmt.Sprintf("%s (%d-%d)", prompt, lo, hi), func(s string) (int, error) {
		n, err := strconv.Atoi(s)
		if err != nil || n < lo || n > hi {
			return 0, fmt.Errorf("нужно целое число от %d до %d", lo, hi)
		}
		return n, nil
	})
}

func (a *App) askYesNo(prompt string) (bool, error) {
	return ask(a, prompt+" (д/н)", func(s string) (bool, error) {
		switch strings.ToLower(s) {
		case "д", "да", "y", "yes":
			return true, nil
		case "н", "нет", "n", "no":
			return false, nil
		}
		return false, errors.New("ответьте д или н")
	})
}

// askID спрашивает номер из списка known и возвращает найденный элемент
func askID[K ~int, V any](a *App, prompt string, known map[K]V) (V, error) {
	return ask(a, prompt, func(s string) (V, error) {
		n, err := strconv.Atoi(s)
		if v, ok := known[K(n)]; err == nil && ok {
			return v, nil
		}
		var zero V
		return zero, errors.New("такого номера нет в списке")
	})
}

// number печатает число без лишних нулей
func number(x float64) string {
	return strconv.FormatFloat(x, 'f', -1, 64)
}
