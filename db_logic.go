package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/*************** Размеры ****************/
var (
	winW float32 = 1280
	winH float32 = 800

	dialogW float32 = 450
	dialogH float32 = 350

	editDlgW   float32 = 420
	editDlgH   float32 = 160
	renameDlgW float32 = 420
	renameDlgH float32 = 160
	newRecDlgW float32 = 520
	newRecDlgH float32 = 360
)

/*************** Иконки *****************/
var pencilSVG = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
<path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04a1.003 1.003 0 0 0 0-1.42L18.37 3.29a1.003 1.003 0 0 0-1.42 0l-1.83 1.83 3.75 3.75 1.84-1.83z" fill="#000"/>
</svg>`)
var pencilIcon = theme.NewThemedResource(fyne.NewStaticResource("pencil.svg", pencilSVG))

/*************** Кнопка-иконка общего назначения **********/
type IconAction struct {
	widget.BaseWidget
	icon    fyne.Resource
	onTap   func()
	hovered bool
	minSize fyne.Size
}

func NewIconAction(icon fyne.Resource, minCell fyne.Size, onTap func()) *IconAction {
	w := &IconAction{icon: icon, onTap: onTap, minSize: minCell}
	w.ExtendBaseWidget(w)
	return w
}
func (i *IconAction) SetOnTapped(f func()) { i.onTap = f }
func (i *IconAction) Tapped(*fyne.PointEvent) {
	if i.onTap != nil {
		i.onTap()
	}
}
func (i *IconAction) MouseIn(*desktop.MouseEvent)    { i.hovered = true; i.Refresh() }
func (i *IconAction) MouseMoved(*desktop.MouseEvent) {}
func (i *IconAction) MouseOut()                      { i.hovered = false; i.Refresh() }

func (i *IconAction) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.HoverColor())
	bg.Hide()
	btn := widget.NewButtonWithIcon("", i.icon, func() {
		if i.onTap != nil {
			i.onTap()
		}
	})
	btn.Importance = widget.LowImportance
	cont := container.NewMax(bg, btn)
	return &iconActionRenderer{owner: i, bg: bg, btn: btn, objs: []fyne.CanvasObject{cont}, cell: i.minSize}
}

type iconActionRenderer struct {
	owner *IconAction
	bg    *canvas.Rectangle
	btn   *widget.Button
	objs  []fyne.CanvasObject
	cell  fyne.Size
}

func (r *iconActionRenderer) Layout(s fyne.Size) {
	w, h := s.Width, s.Height
	if r.cell.Width > w {
		w = r.cell.Width
	}
	if r.cell.Height > h {
		h = r.cell.Height
	}
	r.bg.Resize(fyne.NewSize(w, h))
	r.btn.Resize(fyne.NewSize(w, h))
}
func (r *iconActionRenderer) MinSize() fyne.Size { return r.cell }
func (r *iconActionRenderer) Refresh() {
	if r.owner.hovered {
		r.bg.Show()
	} else {
		r.bg.Hide()
	}
	r.btn.SetIcon(r.owner.icon)
	r.btn.Refresh()
}
func (r *iconActionRenderer) Destroy()                     {}
func (r *iconActionRenderer) Objects() []fyne.CanvasObject { return r.objs }

/*************** Entry с поддержкой Esc **********/
type EscEntry struct {
	widget.Entry
	OnEsc func()
}

func NewEscEntry() *EscEntry {
	e := &EscEntry{}
	e.ExtendBaseWidget(e)
	return e
}
func (e *EscEntry) TypedKey(ev *fyne.KeyEvent) {
	if ev.Name == fyne.KeyEscape {
		if e.OnEsc != nil {
			e.OnEsc()
		}
		return
	}
	e.Entry.TypedKey(ev)
}

/*************** IdCell — статичное поле крестика + центрированный ID **********/
type IdCell struct {
	widget.BaseWidget
	text        string
	onDelete    func()
	hover       bool
	lastSize    fyne.Size
	delPos      fyne.Position
	delSize     fyne.Size
	iconRes     fyne.Resource
	iconPadding float32
}

func NewIdCell() *IdCell {
	c := &IdCell{
		text:        "",
		iconRes:     theme.CancelIcon(),
		iconPadding: 2,
		delSize:     fyne.NewSize(28, 28), // увеличенный хитбокс
	}
	c.ExtendBaseWidget(c)
	return c
}
func (c *IdCell) SetText(s string)     { c.text = s; c.Refresh() }
func (c *IdCell) SetOnDelete(f func()) { c.onDelete = f }

func (c *IdCell) MouseIn(*desktop.MouseEvent)    { c.hover = true; c.Refresh() }
func (c *IdCell) MouseMoved(*desktop.MouseEvent) {}
func (c *IdCell) MouseOut()                      { c.hover = false; c.Refresh() }
func (c *IdCell) Tapped(ev *fyne.PointEvent) {
	if c.onDelete == nil {
		return
	}
	p := ev.Position
	if p.X >= c.delPos.X && p.Y >= c.delPos.Y && p.X <= c.delPos.X+c.delSize.Width && p.Y <= c.delPos.Y+c.delSize.Height {
		c.onDelete()
	}
}
func (c *IdCell) CreateRenderer() fyne.WidgetRenderer {
	txt := canvas.NewText(c.text, theme.ForegroundColor())
	txt.Alignment = fyne.TextAlignCenter
	txt.TextStyle = fyne.TextStyle{Bold: true}
	txt.TextSize = theme.TextSize() + 2

	icon := canvas.NewImageFromResource(c.iconRes)
	icon.FillMode = canvas.ImageFillContain

	bgHot := canvas.NewRectangle(theme.HoverColor())
	bgHot.Hide()

	cont := container.NewWithoutLayout(txt, bgHot, icon)
	return &idCellRenderer{
		owner: c,
		txt:   txt,
		icon:  icon,
		hot:   bgHot,
		objs:  []fyne.CanvasObject{cont},
	}
}

type idCellRenderer struct {
	owner *IdCell
	txt   *canvas.Text
	icon  *canvas.Image
	hot   *canvas.Rectangle
	objs  []fyne.CanvasObject
}

func (r *idCellRenderer) Layout(s fyne.Size) {
	r.owner.lastSize = s
	pad := float32(theme.Padding())

	// Геометрия крестика справа
	delW, delH := r.owner.delSize.Width, r.owner.delSize.Height
	if delH > s.Height-2*pad {
		delH = s.Height - 2*pad
		if delH < 20 {
			delH = 20
		}
	}
	if delW > delH {
		delW = delH
	}
	x := s.Width - delW - pad
	y := (s.Height - delH) / 2
	r.owner.delPos = fyne.NewPos(x, y)
	r.icon.Resize(fyne.NewSize(delW-r.owner.iconPadding*2, delH-r.owner.iconPadding*2))
	r.icon.Move(fyne.NewPos(x+r.owner.iconPadding, y+r.owner.iconPadding))

	// Центрирование текста ID в области без зоны крестика
	workW := s.Width - delW - 2*pad
	if workW < 10 {
		workW = 10
	}
	r.txt.Text = r.owner.text
	r.txt.TextSize = theme.TextSize() + 2
	r.txt.TextStyle = fyne.TextStyle{Bold: true}
	txtMin := r.txt.MinSize()
	tx := pad + (workW-txtMin.Width)/2
	if tx < pad {
		tx = pad
	}
	ty := (s.Height - txtMin.Height) / 2
	r.txt.Resize(txtMin)
	r.txt.Move(fyne.NewPos(tx, ty))

	// Подсветка зоны клика
	r.hot.Resize(fyne.NewSize(delW, delH))
	r.hot.Move(fyne.NewPos(x, y))
}

func (r *idCellRenderer) MinSize() fyne.Size {
	h := float32(theme.TextSize()+2) + 2*theme.Padding()
	w := float32(64)
	if w < r.owner.delSize.Width+60 {
		w = r.owner.delSize.Width + 60
	}
	return fyne.NewSize(w, h)
}
func (r *idCellRenderer) Refresh() {
	if r.owner.hover {
		r.icon.Show()
		r.hot.Show()
	} else {
		r.icon.Hide()
		r.hot.Hide()
	}
	r.txt.Text = r.owner.text
	canvas.Refresh(r.txt)
	canvas.Refresh(r.icon)
	canvas.Refresh(r.hot)
}
func (r *idCellRenderer) Destroy()                     {}
func (r *idCellRenderer) Objects() []fyne.CanvasObject { return r.objs }

/*************** Вспомогательные **********/
func getCSVFiles() []string {
	var files []string
	items, _ := os.ReadDir(".")
	for _, it := range items {
		if it.IsDir() {
			continue
		}
		n := it.Name()
		if strings.HasPrefix(n, ".") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(n), ".csv") {
			files = append(files, n)
		}
	}
	sort.Strings(files)
	return files
}

// Перенумерация ID (первая колонка) 1..N после удаления
func renumberIDs(data [][]string) {
	if len(data) == 0 {
		return
	}
	for i := 1; i < len(data); i++ {
		data[i][0] = strconv.Itoa(i)
	}
}

// Разбор списка колонок: "name, age, city" или "name age city"
func parseColumns(colsRaw string) []string {
	cols := []string{}
	if strings.Contains(colsRaw, ",") {
		parts := strings.Split(colsRaw, ",")
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t != "" {
				cols = append(cols, t)
			}
		}
	} else {
		for _, p := range strings.Fields(colsRaw) {
			t := strings.TrimSpace(p)
			if t != "" {
				cols = append(cols, t)
			}
		}
	}
	return cols
}

/*************** Приложение **********/
func main() {
	myApp := app.New()
	myApp.Settings().SetTheme(&forestTheme{})

	win := myApp.NewWindow("CSV DB Manager")
	win.Resize(fyne.NewSize(winW, winH))

	tableListData := binding.NewStringList()
	_ = tableListData.Set(getCSVFiles())

	var current [][]string
	status := widget.NewLabel("Добро пожаловать в CSV DB Manager!")
	var selected string

	var updateTable func([][]string, string)

	// Глобальные флаги и горячие клавиши для диалогов
	var activeDlg *dialog.ConfirmDialog
	var onEnter func() // действие по Enter для текущего диалога

	var editingCell bool
	var editingCellID widget.TableCellID

	// ширина ID-колонки
	idColWidth := float32(64)

	// Таблица: +1 «виртуальная» строка для плюса в колонке 0
	dataTable := widget.NewTable(
		func() (int, int) {
			if len(current) == 0 {
				return 0, 0
			}
			return len(current) + 1, len(current[0])
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(myApp.Settings().Theme().Color(theme.ColorNameInputBackground, myApp.Settings().ThemeVariant()))
			lbl := widget.NewLabel("")
			idCell := NewIdCell()
			plusBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), nil)
			plusBtn.Importance = widget.LowImportance
			plusCenter := container.NewCenter(plusBtn)
			plusCenter.Hide()
			content := container.NewStack(lbl, idCell)
			return container.NewMax(bg, content, plusCenter)
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			if len(current) == 0 {
				return
			}
			con := obj.(*fyne.Container)
			content := con.Objects[1].(*fyne.Container)
			plusCenter := con.Objects[2].(*fyne.Container)

			lbl := content.Objects[0].(*widget.Label)
			idCell := content.Objects[1].(*IdCell)

			plusCenter.Hide()
			idCell.Hide()
			lbl.Show()

			// Плюс — последняя виртуальная строка в колонке 0
			if id.Row == len(current) && id.Col == 0 {
				lbl.Hide()
				idCell.Hide()
				plusCenter.Show()
				plusBtn := plusCenter.Objects[0].(*widget.Button)
				plusBtn.OnTapped = func() {
					showCreateDialog(win, &activeDlg, &onEnter, selected, current, func(updated [][]string) {
						updateTable(updated, selected)
					})
				}
				return
			}

			// Заполнение текста
			if id.Row < len(current) && id.Col < len(current[id.Row]) {
				lbl.SetText(current[id.Row][id.Col])
			} else {
				lbl.SetText("")
			}

			// ID-колонка с крестиком и центрированным текстом
			if id.Col == 0 && id.Row > 0 && id.Row < len(current) {
				lbl.Hide()
				idCell.Show()
				idCell.SetText(current[id.Row][0])
				rowIndex := id.Row
				idCell.SetOnDelete(func() {
					// Диалог подтверждения удаления «Да/Нет»
					text := widget.NewLabel(fmt.Sprintf("Удалить запись с id %s?", current[rowIndex][0]))
					doDelete := func() {
						// Удаляем строку и перенумеровываем id
						newData := make([][]string, 0, len(current)-1)
						for r := range current {
							if r == rowIndex {
								continue
							}
							newData = append(newData, current[r])
						}
						renumberIDs(newData)
						if err := saveTableData(selected, newData); err != nil {
							dialog.ShowError(err, win)
							return
						}
						updateTable(newData, selected)
					}
					dlg := dialog.NewCustomConfirm("Удалить запись", "Да", "Нет", container.NewPadded(text), func(ok bool) {
						if ok {
							doDelete()
						}
						activeDlg = nil
						onEnter = nil
					}, win)
					dlg.Resize(fyne.NewSize(dialogW, dialogH))
					activeDlg = dlg
					onEnter = func() { doDelete() }
					dlg.Show()
				})
				return
			}
		},
	)

	// Глобальная обработка Esc/Enter для активного диалога
	win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if activeDlg == nil {
			return
		}
		switch ev.Name {
		case fyne.KeyEscape:
			activeDlg.Dismiss()
			activeDlg = nil
			onEnter = nil
			if editingCell {
				dataTable.Unselect(editingCellID)
				editingCell = false
			}
		case fyne.KeyReturn, fyne.KeyEnter:
			if onEnter != nil {
				onEnter()
				activeDlg.Dismiss()
				activeDlg = nil
				onEnter = nil
			}
		}
	})

	// Редактирование по ЛКМ
	dataTable.OnSelected = func(id widget.TableCellID) {
		if selected == "" || len(current) == 0 {
			return
		}

		// Переименование заголовков (кроме id)
		if id.Row == 0 {
			if id.Col == 0 {
				inf := dialog.NewInformation("Переименование", "Колонку id переименовывать нельзя", win)
				inf.Resize(fyne.NewSize(dialogW, dialogH))
				inf.Show()
				dataTable.Unselect(id)
				return
			}
			old := current[0][id.Col]
			entry := NewEscEntry()
			entry.SetText(old)
			var dlg *dialog.ConfirmDialog
			commit := func() {
				newVal := strings.TrimSpace(entry.Text)
				if newVal == "" {
					newVal = old
				}
				current[0][id.Col] = newVal
				if err := saveTableData(selected, current); err != nil {
					current[0][id.Col] = old
					dialog.ShowError(err, win)
				} else {
					updateTable(current, selected)
					status.SetText(fmt.Sprintf("Переименован заголовок колонки %d", id.Col))
				}
				dataTable.Unselect(id)
			}
			entry.OnSubmitted = func(string) {
				commit()
				if dlg != nil {
					dlg.Dismiss()
				}
				activeDlg = nil
				onEnter = nil
			}
			entry.OnEsc = func() {
				dataTable.Unselect(id)
				if dlg != nil {
					dlg.Dismiss()
				}
				activeDlg = nil
				onEnter = nil
			}
			dlg = dialog.NewCustomConfirm("Переименовать колонку", "Сохранить", "Отмена", container.NewPadded(entry), func(ok bool) {
				if ok {
					commit()
				} else {
					dataTable.Unselect(id)
				}
				activeDlg = nil
				onEnter = nil
			}, win)
			dlg.Resize(fyne.NewSize(renameDlgW, renameDlgH))
			activeDlg = dlg
			onEnter = func() { commit() }
			dlg.Show()
			return
		}

		// «Плюсовая» строка игнорируется — у неё отдельная кнопка
		if id.Row >= len(current) {
			return
		}
		// колонка id не редактируется
		if id.Col == 0 {
			inf := dialog.NewInformation("Редактирование", "Колонку id редактировать нельзя", win)
			inf.Resize(fyne.NewSize(dialogW, dialogH))
			inf.Show()
			dataTable.Unselect(id)
			return
		}
		if id.Col < 0 || id.Col >= len(current[id.Row]) {
			return
		}

		old := current[id.Row][id.Col]
		entry := NewEscEntry()
		entry.SetText(old)
		var dlg *dialog.ConfirmDialog

		commit := func() {
			newVal := entry.Text
			current[id.Row][id.Col] = newVal
			if err := saveTableData(selected, current); err != nil {
				current[id.Row][id.Col] = old
				dialog.ShowError(err, win)
			} else {
				updateTable(current, selected)
				status.SetText(fmt.Sprintf("Изменено row %d col %d", id.Row, id.Col))
			}
			dataTable.Unselect(id)
		}

		entry.OnSubmitted = func(string) {
			commit()
			if dlg != nil {
				dlg.Dismiss()
			}
			activeDlg = nil
			onEnter = nil
			editingCell = false
		}
		entry.OnEsc = func() {
			dataTable.Unselect(id)
			if dlg != nil {
				dlg.Dismiss()
			}
			activeDlg = nil
			onEnter = nil
			editingCell = false
		}

		dlg = dialog.NewCustomConfirm("Редактировать ячейку", "Сохранить", "Отмена", container.NewPadded(entry), func(ok bool) {
			if ok {
				commit()
			} else {
				dataTable.Unselect(id)
			}
			activeDlg = nil
			onEnter = nil
			editingCell = false
		}, win)
		dlg.Resize(fyne.NewSize(editDlgW, editDlgH))
		activeDlg = dlg
		onEnter = func() { commit() }
		editingCell = true
		editingCellID = id
		dlg.Show()
	}

	// Обновление таблицы и статуса
	updateTable = func(data [][]string, name string) {
		current = data
		if len(data) > 0 && len(data[0]) > 0 {
			dataTable.SetColumnWidth(0, idColWidth)
			for i := 1; i < len(data[0]); i++ {
				dataTable.SetColumnWidth(i, 220)
			}
		}
		dataTable.Refresh()
		if len(data) > 1 {
			status.SetText(fmt.Sprintf("Таблица %s загружена Строк %d", name, len(data)-1))
		} else if len(data) == 1 {
			status.SetText(fmt.Sprintf("Таблица %s пуста", name))
		} else {
			status.SetText(fmt.Sprintf("Таблица %s пуста или не найдена", name))
		}
	}

	/*************** Список таблиц ***************/
	var list *widget.List
	cellSize := fyne.NewSize(28, 28)

	makeListRow := func() fyne.CanvasObject {
		name := widget.NewLabel("")
		open := NewIconAction(theme.FolderIcon(), cellSize, nil)
		copy := NewIconAction(theme.ContentCopyIcon(), cellSize, nil)
		ren := NewIconAction(pencilIcon, cellSize, nil)
		del := NewIconAction(theme.DeleteIcon(), cellSize, nil) // удаление файла таблицы
		actions := container.NewHBox(open, copy, ren, del)
		actions.Hide()
		return container.NewHBox(name, layout.NewSpacer(), actions)
	}

	list = widget.NewListWithData(
		tableListData,
		makeListRow,
		func(it binding.DataItem, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			nameLabel := row.Objects[0].(*widget.Label)
			actions := row.Objects[2].(*fyne.Container)

			val, _ := it.(binding.String).Get()
			fn := val
			nameLabel.SetText(fn)

			open := actions.Objects[0].(*IconAction)
			copyAct := actions.Objects[1].(*IconAction)
			renAct := actions.Objects[2].(*IconAction)
			delAct := actions.Objects[3].(*IconAction)

			open.SetOnTapped(func() {
				abs, err := filepath.Abs(fn)
				if err != nil {
					dialog.ShowError(err, win)
					return
				}
				if err := openFile(abs); err != nil {
					dialog.ShowError(err, win)
				}
			})

			copyAct.SetOnTapped(func() {
				entry := NewEscEntry()
				base := strings.TrimSuffix(fn, filepath.Ext(fn))
				entry.SetText(base + "_copy.csv")

				commitCopy := func() {
					newName := entry.Text
					if !strings.HasSuffix(newName, ".csv") {
						newName += ".csv"
					}
					if err := copyFile(fn, newName); err != nil {
						dialog.ShowError(err, win)
						return
					}
					_ = tableListData.Set(getCSVFiles())
					list.Refresh()
					status.SetText(fmt.Sprintf("Таблица %s скопирована в %s", fn, newName))
				}

				entry.OnSubmitted = func(string) {
					commitCopy()
					if activeDlg != nil {
						activeDlg.Dismiss()
					}
					activeDlg = nil
					onEnter = nil
				}
				entry.OnEsc = func() {
					if activeDlg != nil {
						activeDlg.Dismiss()
					}
					activeDlg = nil
					onEnter = nil
				}

				dlg := dialog.NewCustomConfirm("Копировать таблицу", "Копировать", "Отмена", entry, func(ok bool) {
					if ok {
						commitCopy()
					}
					activeDlg = nil
					onEnter = nil
				}, win)
				dlg.Resize(fyne.NewSize(dialogW, dialogH))
				activeDlg = dlg
				onEnter = func() { commitCopy() }
				dlg.Show()
			})

			renAct.SetOnTapped(func() {
				entry := NewEscEntry()
				entry.SetText(fn)

				commitRename := func() {
					newName := entry.Text
					if !strings.HasSuffix(newName, ".csv") {
						newName += ".csv"
					}
					if err := os.Rename(fn, newName); err != nil {
						dialog.ShowError(err, win)
						return
					}
					_ = tableListData.Set(getCSVFiles())
					list.Refresh()
					status.SetText(fmt.Sprintf("Таблица %s переименована в %s", fn, newName))
				}

				entry.OnSubmitted = func(string) {
					commitRename()
					if activeDlg != nil {
						activeDlg.Dismiss()
					}
					activeDlg = nil
					onEnter = nil
				}
				entry.OnEsc = func() {
					if activeDlg != nil {
						activeDlg.Dismiss()
					}
					activeDlg = nil
					onEnter = nil
				}

				dlg := dialog.NewCustomConfirm("Переименовать таблицу", "Переименовать", "Отмена", container.NewPadded(entry), func(ok bool) {
					if ok {
						commitRename()
					}
					activeDlg = nil
					onEnter = nil
				}, win)
				dlg.Resize(fyne.NewSize(renameDlgW, renameDlgH))
				activeDlg = dlg
				onEnter = func() { commitRename() }
				dlg.Show()
			})

			delAct.SetOnTapped(func() {
				text := widget.NewLabel(fmt.Sprintf("Удалить таблицу %s?", fn))
				commitDelete := func() {
					if err := deleteTable(fn); err != nil {
						dialog.ShowError(err, win)
						return
					}
					_ = tableListData.Set(getCSVFiles())
					list.Refresh()
					status.SetText(fmt.Sprintf("Таблица %s удалена", fn))
					if selected == fn {
						selected = ""
						updateTable(nil, "")
					}
				}
				dlg := dialog.NewCustomConfirm("Удалить таблицу", "Да", "Нет", container.NewPadded(text), func(ok bool) {
					if ok {
						commitDelete()
					}
					activeDlg = nil
					onEnter = nil
				}, win)
				dlg.Resize(fyne.NewSize(dialogW, dialogH))
				activeDlg = dlg
				onEnter = func() { commitDelete() }
				dlg.Show()
			})

			if fn == selected {
				actions.Show()
			} else {
				actions.Hide()
			}
		},
	)

	/*************** Команды ***************/
	commandsDesc := "CREATE <table> <col1,col2..> - создать таблицу с n-колонок. | FIND <table> <column> <value> - найти нужное значение в выбранной таблице и колонке."
	cmdEntry := widget.NewEntry()
	cmdEntry.SetPlaceHolder("Введите команду create или find ...")
	cmdEntry.OnSubmitted = func(text string) {
		cmd, table, args, err := parseQuery(text)
		if err != nil {
			status.SetText("Ошибка парсинга " + err.Error())
			return
		}
		switch cmd {
		case "create":
			// args — это уже разложенные названия колонок
			if table == "" || len(args) == 0 {
				status.SetText("create требует имя таблицы и список колонок")
				break
			}
			headers := append([]string{"id"}, args...)
			newData := make([][]string, 0, 1)
			newData = append(newData, headers)
			filename := table + ".csv"
			if err := saveTableData(filename, newData); err != nil {
				status.SetText("Ошибка " + err.Error())
				break
			}
			selected = filename
			_ = tableListData.Set(getCSVFiles())
			updateTable(newData, selected)
			list.Refresh()
			status.SetText("Таблица " + table + " создана: " + strings.Join(args, ", "))
		case "find":
			if len(args) < 2 {
				status.SetText("find требует колонку и значение")
				break
			}
			if data, err := findRecord(table, args[0], args[1]); err != nil {
				status.SetText("Ошибка " + err.Error())
			} else {
				selected = table + ".csv"
				updateTable(data, selected)
				list.Refresh()
			}
		default:
			status.SetText("Неизвестная команда " + cmd)
		}
		cmdEntry.SetText("")
	}

	// Левая панель 20% — список; правая 80% — таблица + команды снизу
	leftBg := canvas.NewRectangle(myApp.Settings().Theme().Color(theme.ColorNameInputBackground, myApp.Settings().ThemeVariant()))
	leftPanel := widget.NewCard("Таблицы", "", container.NewMax(leftBg, list))

	commandsBox := widget.NewCard("Команды", commandsDesc, container.NewPadded(cmdEntry))
	rightPanel := container.NewBorder(nil, container.NewVBox(commandsBox, status), nil, nil, dataTable)

	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.2 // 20% слева, 80% справа

	win.SetContent(split)

	// Выбор таблицы слева
	list.OnSelected = func(id widget.ListItemID) {
		fn, err := tableListData.GetValue(id)
		if err != nil {
			status.SetText("Ошибка выбора " + err.Error())
			return
		}
		selected = fn
		if data, err := readTableData(fn); err != nil {
			status.SetText("Ошибка " + err.Error())
			updateTable(nil, fn)
		} else {
			updateTable(data, fn)
		}
		list.Refresh()
	}

	// Запуск
	win.ShowAndRun()
}

/*************** Диалог создания записи ***************/
func showCreateDialog(
	win fyne.Window,
	activeDlg **dialog.ConfirmDialog,
	onEnter *func(),
	selected string,
	current [][]string,
	onSaved func([][]string),
) {
	// Информ-диалог фиксированного размера
	showSizedInfo := func(title, msg string) {
		lbl := widget.NewLabel(msg)
		d := dialog.NewCustom(title, "OK", container.NewPadded(lbl), win)
		d.Resize(fyne.NewSize(dialogW, dialogH))
		d.Show()
	}

	if selected == "" || len(current) == 0 {
		showSizedInfo("Создание", "Сначала выберите таблицу")
		return
	}
	headers := current[0][1:]
	if len(headers) == 0 {
		showSizedInfo("Создание", "Нет редактируемых колонок")
		return
	}

	fields := make([]*EscEntry, len(headers))
	form := container.NewVBox()
	for i, h := range headers {
		lbl := widget.NewLabel(h)
		entry := NewEscEntry()
		entry.SetPlaceHolder(h)
		entry.Wrapping = fyne.TextWrapOff
		entry.SetMinRowsVisible(1)
		fields[i] = entry
		form.Add(container.NewBorder(nil, nil, lbl, nil, entry))
	}

	info := widget.NewLabel("")
	updateInfo := func() {
		filled := 0
		for _, f := range fields {
			if strings.TrimSpace(f.Text) != "" {
				filled++
			}
		}
		info.SetText(fmt.Sprintf("Заполнены %d полей из %d необходимых", filled, len(fields)))
	}
	for _, f := range fields {
		f.OnChanged = func(string) { updateInfo() }
	}
	updateInfo()

	content := container.NewVBox(form, info)

	var dlg *dialog.ConfirmDialog
	commit := func() {
		values := make([]string, len(fields))
		for i, f := range fields {
			v := strings.TrimSpace(f.Text)
			if v == "" {
				updateInfo()
				return
			}
			values[i] = v
		}
		table := strings.TrimSuffix(selected, ".csv")
		if err := insertRecord(table, values); err != nil {
			dialog.ShowError(err, win)
			return
		}
		if data, e := readTableData(selected); e == nil {
			onSaved(data)
		}
	}

	for _, f := range fields {
		ff := f
		ff.OnSubmitted = func(string) {
			before := info.Text
			commit()
			after := info.Text
			if before == after {
				if dlg != nil {
					dlg.Dismiss()
				}
				*activeDlg = nil
				*onEnter = nil
			}
		}
		ff.OnEsc = func() {
			if dlg != nil {
				dlg.Dismiss()
			}
			*activeDlg = nil
			*onEnter = nil
		}
	}

	dlg = dialog.NewCustomConfirm("Новая запись", "Сохранить", "Отмена", content, func(ok bool) {
		if ok {
			all := true
			for _, f := range fields {
				if strings.TrimSpace(f.Text) == "" {
					all = false
					break
				}
			}
			if !all {
				updateInfo()
				return
			}
			commit()
		}
		*activeDlg = nil
		*onEnter = nil
	}, win)
	dlg.Resize(fyne.NewSize(newRecDlgW, newRecDlgH))
	*activeDlg = dlg
	*onEnter = func() { commit() }
	dlg.Show()
}

/*************** Парсер команд ***************/
func parseQuery(q string) (cmd, table string, args []string, err error) {
	parts := strings.Fields(q)
	if len(parts) == 0 {
		return "", "", nil, fmt.Errorf("пустой запрос")
	}
	cmd = strings.ToLower(parts[0])
	if len(parts) < 2 {
		return "", "", nil, fmt.Errorf("не указано имя таблицы")
	}
	table = parts[1]

	switch cmd {
	case "create":
		if len(parts) < 3 {
			return "", "", nil, fmt.Errorf("create требует список колонок")
		}
		colsRaw := strings.Join(parts[2:], " ")
		args = parseColumns(colsRaw)
		if len(args) == 0 {
			return "", "", nil, fmt.Errorf("не найдены названия колонок")
		}
	case "find":
		if len(parts) < 4 {
			return "", "", nil, fmt.Errorf("find: укажите колонку и значение")
		}
		col := parts[2]
		val := strings.Join(parts[3:], " ")
		args = []string{col, val}
	default:
		// поддерживаем только create/find в этой строке команд
	}
	return
}
