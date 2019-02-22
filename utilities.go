package tooling

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"unicode"
)

func ParseCommandLine(s string) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)

		}
	}

	fields := strings.FieldsFunc(s, f)

	for i, v := range fields {
		if v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'' {
			fields[i] = v[1 : len(v)-1]
		}
	}

	return fields
}

func ExecuteAndPipeCommandLine(line string, prefix string, quiet bool) error {
	parts := ParseCommandLine(line)
	c := exec.Command(parts[0], parts[1:]...)
	soReader, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	seReader, err := c.StderrPipe()
	if err != nil {
		return err
	}

	const maxCapacity = 1024*1024

	soScanner := bufio.NewScanner(soReader)
	soBuffer := make([]byte, maxCapacity)
	soScanner .Buffer(soBuffer, maxCapacity)
	go func() {
		for soScanner.Scan() {
			if !quiet {
				fmt.Printf("%s%s\n", prefix, soScanner.Text())
			}
		}
	}()

	seScanner := bufio.NewScanner(seReader)
	seBuffer := make([]byte, maxCapacity)
	seScanner .Buffer(seBuffer, maxCapacity)
	go func() {
		for seScanner.Scan() {
			if !quiet {
				fmt.Printf("%s%s\n", prefix, seScanner.Text())
			}
		}
	}()

	err = c.Start()
	if err != nil {
		return err
	}

	err = c.Wait()
	if err != nil {
		return err
	}

	return nil
}
