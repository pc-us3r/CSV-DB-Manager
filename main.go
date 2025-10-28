package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// --- Блок команд + работа с файлами ---

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
	writer := csv.NewWriter(file)
	header := append([]string{"id"}, columns...)
	if err := writer.Write(header); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func getNextID(tableName string) (int, error) {
	file, err := os.Open(tableName)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, err
	}
	maxID := 0
	for i := 1; i < len(records); i++ {
		if len(records[i]) > 0 {
			id, err := strconv.Atoi(records[i][0])
			if err == nil && id > maxID {
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
	id, err := getNextID(fileName)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)

	headerRecs, _ := readTableData(fileName)
	if len(headerRecs) > 0 && len(fieldValues) != len(headerRecs[0])-1 {
		return fmt.Errorf("ошибка: неверное количество полей. Ожидалось %d, получено %d", len(headerRecs[0])-1, len(fieldValues))
	}

	record := append([]string{strconv.Itoa(id)}, fieldValues...)
	err = writer.Write(record)
	writer.Flush()
	return err
}

func readTableData(tableName string) ([][]string, error) {
	file, err := os.Open(tableName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	return reader.ReadAll()
}

func updateRecord(tableName string, id string, fieldValues []string) error {
	fileName := tableName
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}
	records, err := readTableData(fileName)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return errors.New("таблица пуста")
	}

	header, data := records[0], records[1:]
	if len(fieldValues) != len(header)-1 {
		return fmt.Errorf("несоответствие количества полей: ожидалось %d, передано %d", len(header)-1, len(fieldValues))
	}

	updated := false
	for i, row := range data {
		if len(row) > 0 && row[0] == id {
			records[i+1] = append([]string{id}, fieldValues...)
			updated = true
			break
		}
	}

	if !updated {
		return fmt.Errorf("запись с id=%s не найдена", id)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.WriteAll(records)
	writer.Flush()
	return writer.Error()
}

func findRecord(tableName string, columnName string, value string) ([][]string, error) {
	fileName := tableName
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}
	records, err := readTableData(fileName)
	if err != nil {
		return nil, err
	}
	if len(records) < 1 {
		return nil, errors.New("таблица пуста")
	}

	header := records[0]
	colIndex := -1
	for i, colName := range header {
		if strings.EqualFold(colName, columnName) {
			colIndex = i
			break
		}
	}
	if colIndex == -1 {
		return nil, fmt.Errorf("колонка '%s' не найдена", columnName)
	}

	result := [][]string{header}
	for _, record := range records[1:] {
		if len(record) > colIndex && strings.EqualFold(record[colIndex], value) {
			result = append(result, record)
		}
	}

	if len(result) == 1 {
		return nil, fmt.Errorf("записи со значением '%s' не найдены", value)
	}
	return result, nil
}

func deleteRecord(tableName string, id string) error {
	fileName := tableName
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}
	records, err := readTableData(fileName)
	if err != nil {
		return err
	}
	newRecords := [][]string{}
	found := false
	for _, record := range records {
		if len(record) > 0 && record[0] == id {
			found = true
			continue
		}
		newRecords = append(newRecords, record)
	}
	if !found {
		return fmt.Errorf("запись с id=%s не найдена", id)
	}
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.WriteAll(newRecords)
	writer.Flush()
	return writer.Error()
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
	_, err = io.Copy(destination, source)
	return err
}

func openFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
