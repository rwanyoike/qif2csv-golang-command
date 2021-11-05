package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
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

func qifToCsv(path string, writer csv.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, "!Type:") {
			log.Fatalf("invalid qif header '%s'", line)
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
			log.Printf("unknown field '%s'\n", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)

	qifPath := os.Args[1] // target .qif

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	err := writer.Write([]string{"date", "reference", "note", "amount"})
	if err != nil {
		log.Fatalln(err)
	}

	path, err := filepath.Abs(qifPath)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("parsing '%s'\n", path)
	err = qifToCsv(qifPath, *writer)
	if err != nil {
		log.Fatalln(err)
	}
}
