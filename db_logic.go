package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Блок ПКМ
type TappableLabel struct {
	widget.Label
	OnTappedSecondary func(*fyne.PointEvent)
}

func NewTappableLabel() *TappableLabel {
	label := &TappableLabel{}
	label.ExtendBaseWidget(label)
	return label
}

func (l *TappableLabel) TappedSecondary(e *fyne.PointEvent) {
	if l.OnTappedSecondary != nil {
		l.OnTappedSecondary(e)
	}
}

// --- Блок графической оболочки программы ---

func getCSVFiles() []string {
	var files []string
	items, _ := os.ReadDir(".")
	for _, item := range items {
		if !item.IsDir() && strings.HasSuffix(item.Name(), ".csv") {
			files = append(files, item.Name())
		}
	}
	return files
}

func main() {
	myApp := app.New()
	myApp.Settings().SetTheme(&forestTheme{})

	myWindow := myApp.NewWindow("CSV DB Manager")
	myWindow.Resize(fyne.NewSize(1280, 800))

	tableListData := binding.NewStringList()
	tableListData.Set(getCSVFiles())

	var currentTableData [][]string
	statusLabel := widget.NewLabel("Добро пожаловать в CSV DB Manager!")
	var selectedTableName string

	dataTable := widget.NewTable(
		func() (int, int) {
			if len(currentTableData) == 0 {
				return 0, 0
			}
			return len(currentTableData), len(currentTableData[0])
		},
		func() fyne.CanvasObject {
			cellLabel := widget.NewLabel("Ячейка")
			bg := canvas.NewRectangle(myApp.Settings().Theme().Color(theme.ColorNameInputBackground, myApp.Settings().ThemeVariant()))
			return container.NewMax(bg, cellLabel)
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			cellContainer := o.(*fyne.Container)
			cellLabel := cellContainer.Objects[1].(*widget.Label)
			if i.Row >= len(currentTableData) || i.Col >= len(currentTableData[i.Row]) {
				cellLabel.SetText("")
				return
			}
			cellLabel.SetText(currentTableData[i.Row][i.Col])
		},
	)

	updateTableWidget := func(data [][]string, tableName string) {
		currentTableData = data
		if len(data) > 0 && len(data[0]) > 0 {
			dataTable.SetColumnWidth(0, 50)
			for i := 1; i < len(data[0]); i++ {
				dataTable.SetColumnWidth(i, 200)
			}
		}
		dataTable.Refresh()
		if len(data) > 1 {
			statusLabel.SetText(fmt.Sprintf("Таблица '%s' загружена. Строк: %d.", tableName, len(data)-1))
		} else if len(data) == 1 {
			statusLabel.SetText(fmt.Sprintf("Таблица '%s' пуста.", tableName))
		} else {
			statusLabel.SetText(fmt.Sprintf("Таблица '%s' пуста или не найдена.", tableName))
		}
	}

	tableList := widget.NewListWithData(
		tableListData,
		func() fyne.CanvasObject { return NewTappableLabel() },
		func(i binding.DataItem, o fyne.CanvasObject) {
			label := o.(*TappableLabel)
			val, _ := i.(binding.String).Get()
			label.SetText(val)

			label.OnTappedSecondary = func(e *fyne.PointEvent) {
				fileName, _ := i.(binding.String).Get()

				showItem := fyne.NewMenuItem("Показать таблицу", func() {
					selectedTableName = fileName
					data, err := readTableData(fileName)
					if err != nil {
						statusLabel.SetText("Ошибка: " + err.Error())
						updateTableWidget(nil, fileName)
						return
					}
					updateTableWidget(data, fileName)
				})

				openItem := fyne.NewMenuItem("Открыть таблицу", func() {
					absPath, err := filepath.Abs(fileName)
					if err != nil {
						dialog.ShowError(err, myWindow)
						return
					}
					err = openFile(absPath)
					if err != nil {
						dialog.ShowError(err, myWindow)
					}
				})

				renameItem := fyne.NewMenuItem("Переименовать", func() {
					entry := widget.NewEntry()
					entry.SetText(fileName)
					dialog.ShowCustomConfirm("Переименовать таблицу", "Переименовать", "Отмена", entry, func(ok bool) {
						if !ok {
							return
						}
						newName := entry.Text
						if !strings.HasSuffix(newName, ".csv") {
							newName += ".csv"
						}
						if err := os.Rename(fileName, newName); err != nil {
							dialog.ShowError(err, myWindow)
							return
						}
						tableListData.Set(getCSVFiles())
						statusLabel.SetText(fmt.Sprintf("Таблица '%s' переименована в '%s'", fileName, newName))
					}, myWindow)
				})

				deleteItem := fyne.NewMenuItem("Удалить", func() {
					dialog.ShowConfirm("Удалить таблицу", fmt.Sprintf("Вы уверены, что хотите удалить '%s'?", fileName), func(ok bool) {
						if !ok {
							return
						}
						if err := deleteTable(fileName); err != nil {
							dialog.ShowError(err, myWindow)
							return
						}
						if selectedTableName == fileName {
							updateTableWidget(nil, "")
						}
						tableListData.Set(getCSVFiles())
						statusLabel.SetText(fmt.Sprintf("Таблица '%s' удалена.", fileName))
					}, myWindow)
				})

				copyItem := fyne.NewMenuItem("Копировать", func() {
					entry := widget.NewEntry()
					base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
					entry.SetText(base + "_copy.csv")
					dialog.ShowCustomConfirm("Копировать таблицу", "Копировать", "Отмена", entry, func(ok bool) {
						if !ok {
							return
						}
						newName := entry.Text
						if !strings.HasSuffix(newName, ".csv") {
							newName += ".csv"
						}
						if err := copyFile(fileName, newName); err != nil {
							dialog.ShowError(err, myWindow)
							return
						}
						tableListData.Set(getCSVFiles())
						statusLabel.SetText(fmt.Sprintf("Таблица '%s' скопирована в '%s'", fileName, newName))
					}, myWindow)
				})

				menu := fyne.NewMenu("", showItem, openItem, fyne.NewMenuItemSeparator(), renameItem, copyItem, deleteItem)
				widget.ShowPopUpMenuAtPosition(menu, myWindow.Canvas(), e.AbsolutePosition)
			}
		},
	)

	commandsText := `CREATE - Создание новой таблицы.
  Пример: CREATE users имя,email

INSERT - Добавление новой записи.
  Пример: INSERT users "Иван Петров",ivan@example.com

FIND - Поиск записей по значению в колонке.
  Пример: FIND users email ivan@example.com

UPDATE - Обновление записи по её ID.
  Пример: UPDATE users 1 "Иван Смирнов",ivan.s@example.com

DELETE - Удаление записи по её ID.
  Пример: DELETE users 1`

	commandsLabel := widget.NewLabel(commandsText)

	readMeButton := widget.NewButton("Read Me", func() {
		readmeText := "CSV DB Manager v1 28.10.2025\n\n" +
			"Графический интерфейс создан при помощи Fyne/V2.\n\n" +
			"Вы можете связаться со мной:\n" +
			"TG - @eternalit | Mail - s3cnd0re@mail.ru"

		content := widget.NewMultiLineEntry()
		content.SetText(readmeText)
		content.Wrapping = fyne.TextWrapWord
		content.Disable()

		dlg := dialog.NewCustom("Read Me", "Закрыть", content, myWindow)
		dlg.Resize(fyne.NewSize(450, 350))
		dlg.Show()
	})

	cardContent := container.NewBorder(
		container.NewHBox(layout.NewSpacer(), readMeButton),
		nil, nil, nil,
		container.NewPadded(commandsLabel),
	)

	commandsContainer := container.NewMax(
		canvas.NewRectangle(myApp.Settings().Theme().Color(theme.ColorNameInputBackground, myApp.Settings().ThemeVariant())),
		cardContent,
	)

	tableListContainer := container.NewMax(
		canvas.NewRectangle(myApp.Settings().Theme().Color(theme.ColorNameInputBackground, myApp.Settings().ThemeVariant())),
		tableList,
	)

	inputEntry := widget.NewEntry()
	inputEntry.SetPlaceHolder("Введите команду...")
	inputEntry.OnSubmitted = func(text string) {
		cmd, table, args, err := parseQuery(text)
		if err != nil {
			statusLabel.SetText("Ошибка парсинга: " + err.Error())
			return
		}

		var resultData [][]string
		var resultMsg string

		switch cmd {
		case "create":
			err = createTable(table, args)
			if err == nil {
				resultMsg = fmt.Sprintf("Таблица '%s' создана.", table)
				tableListData.Set(getCSVFiles())
			}
		case "insert":
			err = insertRecord(table, args)
			if err == nil {
				resultMsg = "Запись добавлена."
			}
		case "update":
			err = updateRecord(table, args[0], args[1:])
			if err == nil {
				resultMsg = "Запись обновлена."
			}
		case "delete":
			err = deleteRecord(table, args[0])
			if err == nil {
				resultMsg = "Запись удалена."
			}
		case "find":
			resultData, err = findRecord(table, args[0], args[1])
		default:
			err = fmt.Errorf("неизвестная команда: %s", cmd)
		}

		if err != nil {
			statusLabel.SetText("Ошибка выполнения: " + err.Error())
		} else if resultData != nil {
			selectedTableName = table + ".csv"
			updateTableWidget(resultData, selectedTableName)
		} else {
			statusLabel.SetText(resultMsg)
			if cmd == "insert" || cmd == "update" || cmd == "delete" {
				if selectedTableName == table+".csv" {
					data, _ := readTableData(selectedTableName)
					updateTableWidget(data, selectedTableName)
				}
			}
		}
		inputEntry.SetText("")
	}

	leftPanel := container.NewBorder(widget.NewCard("Таблицы", "", nil), nil, nil, nil, tableListContainer)
	commandsCard := widget.NewCard("Команды", "", commandsContainer)

	rightPanel := container.NewBorder(
		nil,
		container.NewVBox(statusLabel, inputEntry),
		nil,
		nil,
		dataTable,
	)

	split := container.NewHSplit(
		container.NewVSplit(leftPanel, commandsCard),
		rightPanel,
	)
	split.Offset = 0.3

	myWindow.SetContent(split)
	myWindow.ShowAndRun()
}

func parseQuery(query string) (cmd, table string, args []string, err error) {
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return "", "", nil, errors.New("пустой запрос")
	}

	cmd = strings.ToLower(parts[0])
	if len(parts) > 1 {
		table = parts[1]
	} else {
		return "", "", nil, errors.New("не указано имя таблицы")
	}

	switch cmd {
	case "create", "insert":
		if len(parts) < 3 {
			err = fmt.Errorf("команда %s требует аргументы", cmd)
			return
		}
		argsPart := strings.Join(parts[2:], " ")
		r := csv.NewReader(strings.NewReader(argsPart))
		r.TrimLeadingSpace = true
		record, e := r.Read()
		if e != nil {
			err = fmt.Errorf("ошибка парсинга аргументов: %w", e)
			return
		}
		args = record
	case "update":
		if len(parts) < 4 {
			err = errors.New("UPDATE требует id и значения")
			return
		}
		args = append(args, parts[2]) // id
		argsPart := strings.Join(parts[3:], " ")
		r := csv.NewReader(strings.NewReader(argsPart))
		r.TrimLeadingSpace = true
		record, e := r.Read()
		if e != nil {
			err = fmt.Errorf("ошибка парсинга значений для UPDATE: %w", e)
			return
		}
		args = append(args, record...)
	case "delete":
		if len(parts) != 3 {
			err = fmt.Errorf("%s требует имя таблицы и один аргумент", cmd)
			return
		}
		args = []string{parts[2]}
	case "find":
		if len(parts) != 4 {
			err = errors.New("FIND требует имя таблицы, колонку и значение")
			return
		}
		args = parts[2:]
	default:
		err = fmt.Errorf("неизвестная команда: %s", cmd)
	}
	return
}
