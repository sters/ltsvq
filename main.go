package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/Songmu/go-ltsv"
	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	input     *os.File
	output    *os.File
	verbosity bool
	query     string
}

var logger *log.Logger

type nopWriter struct{}

func (*nopWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func parseArgs() (*config, error) {
	config := &config{
		input:     os.Stdin,
		output:    os.Stdout,
		verbosity: false,
	}

	var (
		input     = flag.String("i", "", "input file, default = stdin, skippable")
		output    = flag.String("o", "", "output file, default = stdout, skippable")
		verbosity = flag.Bool("v", false, "verbosity, default = false. if true, say logs")
		query     = flag.String("q", "", "query file, must addition")
		err       error
	)
	flag.Parse()

	if *input != "" {
		config.input, err = os.OpenFile(*input, os.O_RDWR, 0666)
		if err != nil {
			return nil, err
		}
	}

	if *output != "" {
		config.output, err = os.OpenFile(*output, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}
	}

	if *verbosity != false {
		config.verbosity = *verbosity
	}
	setupLogger(config)

	if *query == "" {
		return nil, fmt.Errorf("must satisfied query parameter")
	}
	config.query = *query

	return config, nil
}

func setupLogger(c *config) {
	writer := io.Writer(&nopWriter{})
	if c.verbosity {
		writer = os.Stderr
	}
	logger = log.New(writer, "", log.LstdFlags)
}

func main() {
	config, err := parseArgs()
	if err != nil {
		log.Fatal(err)
	}

	table, err := newLTSVTable()
	if err != nil {
		log.Fatal(err)
	}
	defer table.Close()

	reader := bufio.NewReader(config.input)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}

		err = table.Insert([]byte(strings.TrimSpace(string(line))))
		if err != nil {
			logger.Printf("failed to convert ltsv: %s", err)
		}
	}

	resultSet, err := table.Query(config.query)
	if err != nil {
		log.Fatalf("failed to query: %s", err)
	}

	var outputColumns []string
	for _, r := range resultSet {
		// use another solution
		// b, err := ltsv.Marshal(r)
		// if err != nil {
		// 	log.Fatalf("failed to ltsv marshal: %s", err)
		// }

		if len(outputColumns) == 0 {
			outputColumns = make([]string, 0, len(r))
			for c := range r {
				outputColumns = append(outputColumns, c)
			}
			sort.Strings(outputColumns)
		}

		for _, c := range outputColumns {
			if _, err = config.output.WriteString(c + ":" + r[c] + "\t"); err != nil {
				log.Fatalf("failed to write bytes: %s", err)
			}
		}
		if _, err = config.output.Write([]byte("\n")); err != nil {
			log.Fatalf("failed to write bytes: %s", err)
		}
	}
}

type ltsvTable struct {
	db      *sql.DB
	columns map[string]struct{}
}

func (t *ltsvTable) Close() {
	t.db.Close()
}

func (t *ltsvTable) Insert(rawLTSV []byte) error {
	mapp := map[string]string{}
	err := ltsv.Unmarshal(rawLTSV, &mapp)
	if err != nil {
		return err
	}

	columnNames := make([]string, 0, len(mapp))
	columnBinds := make([]string, 0, len(mapp))
	columnValues := make([]interface{}, 0, len(mapp))
	for k, v := range mapp {
		if _, ok := t.columns[k]; !ok {
			_, err := t.db.Exec(fmt.Sprintf(`ALTER TABLE ltsv ADD COLUMN %s TEXT;`, k))
			if err != nil {
				return err
			}

			t.columns[k] = struct{}{}
		}
		columnNames = append(columnNames, k)
		columnBinds = append(columnBinds, "?")
		columnValues = append(columnValues, v)
	}

	_, err = t.db.Exec(
		fmt.Sprintf(
			`INSERT INTO ltsv (%s) VALUES (%s)`,
			strings.Join(columnNames, ","),
			strings.Join(columnBinds, ","),
		),
		columnValues...,
	)
	if err != nil {
		return err
	}

	return nil
}

func (t *ltsvTable) Query(s string) ([]map[string]string, error) {
	rows, err := t.db.Query(s)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	resultSet := []map[string]string{}
	for rows.Next() {
		type bb = []byte
		values := make([]interface{}, len(columns))
		for i := range values {
			x := []byte{}
			values[i] = &x
		}

		err = rows.Scan(values...)
		if err != nil {
			logger.Printf("cant scan row: %s", err)
		}

		r := map[string]string{}
		for i, v := range values {
			if vv, ok := v.(*[]byte); ok {
				r[columns[i]] = string(*vv)
			}
		}
		resultSet = append(resultSet, r)
	}

	return resultSet, nil
}

func newLTSVTable() (*ltsvTable, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE ltsv (_dummy text);`)
	if err != nil {
		return nil, err
	}

	return &ltsvTable{
		db:      db,
		columns: make(map[string]struct{}),
	}, nil
}
