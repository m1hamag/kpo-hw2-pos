// Консольное приложение ПОС, преподавательская оценка студентов
//
//	go run ./cmd/pos                  меню, данные сохраняются в pos.json
//	go run ./cmd/pos -data my.json    другой файл с данными
//	go run ./cmd/pos -data ""         без сохранения
//	go run ./cmd/pos -demo            сквозной сценарий на заготовленном вводе
package main

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/dig"

	"pos/internal/adapters/primary/console"
	"pos/internal/di"
)

func main() {
	data := flag.String("data", "pos.json", "файл с данными, пустая строка значит без сохранения")
	demo := flag.Bool("demo", false, "прогнать сквозной сценарий на заготовленном вводе, без сохранения")
	flag.Parse()

	cfg := di.Config{DataPath: *data, In: os.Stdin, Out: os.Stdout}
	if *demo {
		cfg.DataPath = ""
		cfg.In = console.DemoInput(os.Stdout)
	}

	container, err := di.NewContainer(cfg)
	if err == nil {
		err = container.Invoke(func(a *console.App) error { return a.Run() })
	}
	if err != nil {
		// dig оборачивает ошибку цепочкой сборки, пользователю нужна только причина
		fmt.Fprintln(os.Stderr, "ошибка:", dig.RootCause(err))
		os.Exit(1)
	}
}
