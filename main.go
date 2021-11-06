package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	Date      string
	Reference string
	Note      string
	Amount    string
}

func findQif(qifPaths *[]string) fs.WalkDirFunc {
	return func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name()[0] == '.' {
			return filepath.SkipDir
		}
		if !d.IsDir() && filepath.Ext(path) == ".qif" {
			*qifPaths = append(*qifPaths, path)
		}
		return nil
	}
}

func fromQifToCsv(qifPath string, writer csv.Writer) error {
	qifFile, err := os.Open(qifPath)
	if err != nil {
		return err
	}
	defer qifFile.Close()

	scanner := bufio.NewScanner(qifFile)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, "!Type:") {
			return fmt.Errorf("'%s' invalid qif header", qifPath)
		}
		break
	}

	entry := Entry{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		code := line[0]
		data := line[1:]
		switch code {
		case 'D':
			date, err := time.Parse("02/01/2006", data)
			if err != nil {
				return err
			}
			entry.Date = date.Format(time.RFC3339)
		case 'N':
			entry.Reference = data
		case 'M':
			entry.Note = data
		case 'T':
			locale := strings.Replace(data, ",", "", -1)
			amount, err := strconv.ParseFloat(locale, 64)
			if err != nil {
				return err
			}
			entry.Amount = fmt.Sprintf("%.2f", amount)
		case '^':
			err := writer.Write([]string{
				entry.Date,
				entry.Reference,
				entry.Note,
				entry.Amount,
			})
			if err != nil {
				return err
			}
			entry = Entry{}
		default:
			log.Printf("unknown field code '%q'", code)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)

	dirPath := os.Args[1] // search .qif
	csvPath := os.Args[2] // output file

	var qifPaths []string
	err := filepath.WalkDir(dirPath, findQif(&qifPaths))
	if err != nil {
		log.Fatal(err)
	}

	csvFile, err := os.Create(csvPath)
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	err = writer.Write([]string{"date", "reference", "note", "amount"})
	if err != nil {
		log.Fatal(err)
	}

	for _, qifPath := range qifPaths {
		fmt.Println("parsing: ", qifPath)
		err := fromQifToCsv(qifPath, *writer)
		if err != nil {
			log.Fatal(err)
		}
	}
}
