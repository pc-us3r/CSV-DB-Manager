package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// --- Работа с CSV-файлами и командами ---

func tableExists(tableName string) bool {
	_, err := os.Stat(tableName)
	return !os.IsNotExist(err)
}

func createTable(tableName string, columns []string) error {
	fileName := tableName
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}
	if tableExists(fileName) {
		return fmt.Errorf("таблица '%s' уже существует", tableName)
	}
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	header := append([]string{"id"}, columns...)
	if err := w.Write(header); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func readHeader(tableName string) ([]string, error) {
	f, err := os.Open(tableName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	return r.Read()
}

func getNextID(tableName string) (int, error) {
	f, err := os.Open(tableName)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	// заголовок
	if _, err := r.Read(); err != nil {
		if errors.Is(err, io.EOF) {
			return 1, nil
		}
		return 0, err
	}

	maxID := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if len(rec) > 0 {
			if id, e := strconv.Atoi(rec[0]); e == nil && id > maxID {
				maxID = id
			}
		}
	}
	return maxID + 1, nil
}

func insertRecord(tableName string, fieldValues []string) error {
	fileName := tableName
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}
	if !tableExists(fileName) {
		return fmt.Errorf("таблица '%s' не найдена", tableName)
	}

	header, err := readHeader(fileName)
	if err != nil {
		return err
	}
	if len(header) > 0 && len(fieldValues) != len(header)-1 {
		return fmt.Errorf("ошибка: неверное количество полей. Ожидалось %d, получено %d", len(header)-1, len(fieldValues))
	}

	id, err := getNextID(fileName)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write(append([]string{strconv.Itoa(id)}, fieldValues...)); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func readTableData(tableName string) ([][]string, error) {
	f, err := os.Open(tableName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	return r.ReadAll()
}

// атомарная замена файла через переименование
func atomicReplace(tempPath, finalPath string) error {
	if err := os.Rename(tempPath, finalPath); err == nil {
		return nil
	}
	if err := os.Remove(finalPath); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

// сохранить все данные таблицы целиком
func saveTableData(tableName string, data [][]string) error {
	fileName := tableName
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}
	dir := filepath.Dir(fileName)
	tmp, err := os.CreateTemp(dir, "csvdb_save_*.csv")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	w := csv.NewWriter(tmp)
	for _, row := range data {
		if err := w.Write(row); err != nil {
			tmp.Close()
			_ = os.Remove(tmpPath)
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := atomicReplace(tmpPath, fileName); err != nil {
		return err
	}
	_ = os.Remove(tmpPath)
	return nil
}

func deleteRecord(tableName string, id string) error {
	fileName := tableName
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}

	in, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer in.Close()

	dir := filepath.Dir(fileName)
	tmp, err := os.CreateTemp(dir, "csvdb_delete_*.csv")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	r := csv.NewReader(in)
	w := csv.NewWriter(tmp)

	header, err := r.Read()
	if err != nil {
		tmp.Close()
		return err
	}
	if err := w.Write(header); err != nil {
		tmp.Close()
		return err
	}

	found := false
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tmp.Close()
			return err
		}
		if len(rec) > 0 && rec[0] == id {
			found = true
			continue
		}
		if err := w.Write(rec); err != nil {
			tmp.Close()
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("запись с id=%s не найдена", id)
	}
	if err := atomicReplace(tmpPath, fileName); err != nil {
		return err
	}
	_ = os.Remove(tmpPath)
	return nil
}

func findRecord(tableName, columnName, value string) ([][]string, error) {
	fileName := tableName
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)

	header, err := r.Read()
	if err == io.EOF {
		return nil, errors.New("таблица пуста")
	}
	if err != nil {
		return nil, err
	}

	colIndex := -1
	for i, col := range header {
		if strings.EqualFold(col, columnName) {
			colIndex = i
			break
		}
	}
	if colIndex == -1 {
		return nil, fmt.Errorf("колонка '%s' не найдена", columnName)
	}

	out := make([][]string, 0, 8)
	out = append(out, header)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) > colIndex && rec[colIndex] == value {
			out = append(out, rec)
		}
	}
	if len(out) == 1 {
		return nil, fmt.Errorf("записи со значением '%s' не найдены", value)
	}
	return out, nil
}

func deleteTable(name string) error {
	filename := name
	if !strings.HasSuffix(filename, ".csv") {
		filename += ".csv"
	}
	return os.Remove(filename)
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}
	if err := destination.Sync(); err != nil {
		return err
	}
	return nil
}

func openFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return fmt.Errorf("не найден xdg-open: %w", err)
		}
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
