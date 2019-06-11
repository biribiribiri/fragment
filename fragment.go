package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gocarina/gocsv"
	"golang.org/x/text/encoding/japanese"
)

func Fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type FoundText struct {
	address   int
	shiftJis  []byte
	utf8Bytes []byte
}

func isAscii(utf8Bytes []byte) bool {
	for _, b := range utf8Bytes {
		if b > 0x7f {
			return false
		}
	}
	return true
}

var jisDecoder = japanese.ShiftJIS.NewDecoder()

// \xFF5F-\xFF9F is half width kana and punctuation  (e.g. ｟ ｠ ｡ ｢ ｣ ､ ･ ｦ ｧ ｨ ｩ )
// \xFF01-\xFF5E is full width alphanumeric (e.g. ０ １ ２ ３)
// \x3000-\x303F japanese punctionation
const CHARS = "[\\p{Han}\\p{Hiragana}\\p{Katakana}" +
	"\x20-\x7E・ー“ΔΘ…○×ΣΛΩ※―”↑★△㎏￥∑↓" +
	"\\x{2E80}-\\x{2FD5}\\x{FF5F}-\\x{FF9F}\\x{FF01}-\\x{FF5E}\\x{3000}-\\x{303F}\\x{31F0}-\\x{31FF}\\x{3220}-\\x{3243}\\x{3280}-\\x{337F}]"

var re = regexp.MustCompile("^" + CHARS + "+$")
var reFilter = regexp.MustCompile(CHARS)
var jpnChar = regexp.MustCompile("[\\p{Han}\\p{Hiragana}\\p{Katakana}]")

//
func parseJIS(data []byte) string {
	utf8Bytes, err := jisDecoder.Bytes(data)
	if err != nil { // Didn't parse as shift-JIS.
		return ""
	}
	// if !utf8.Valid(utf8Bytes) {
	// 	log.Printf("filtered: %s", utf8Bytes)
	// 	return ""
	// }

	if len(data) < 4 {
		// log.Printf("filtered (too short): %s", utf8Bytes)
		return ""
	}
	if isAscii(utf8Bytes) {
		// log.Printf("filtered (only ascii): %s", utf8Bytes)
		return ""
	}
	if !re.Match(utf8Bytes) {
		// invalid := reFilter.ReplaceAllString(string(utf8Bytes), "")
		// log.Printf("filtered (regex): %s [%v]", utf8Bytes, invalid)
		return ""
	}
	if !jpnChar.Match(utf8Bytes) {
		// log.Printf("filtered (no japanese): %s", utf8Bytes)
		return ""
	}

	// if len(re.FindAll(utf8Bytes, 4)) < 4 {
	// 	return ""
	// }

	return string(utf8Bytes)
}

type GameLine struct {
	File           string `csv:"FILE"`
	Offset         int    `csv:"OFFSET"`
	Length         int    `csv:"LENGTH"`
	OriginalText   string `csv:"ORIGINAL_TEXT"`
	TranslatedText string `csv:"TRANSLATED_TEXT"`
}

type ManualFilter struct {
	File        string
	StartOffset int // inclusive
	EndOffset   int // inclusive
}

var manualFilters []*ManualFilter = []*ManualFilter{
	{"DEMOT.PRG", 3799, 30634},
	{"MATCHING.PRG", 1552, 1528523},
	{"GCMNO.PRG", 257, 1062172},
	{"DESKTOPF.PRG", 542, 175730},
	{"GCMNF.PRG", 815, 1582676},
}

func manuallyFiltered(gl *GameLine) bool {
	for _, f := range manualFilters {
		if f.File == gl.File && gl.Offset >= f.StartOffset && gl.Offset <= f.EndOffset {
			log.Printf("filtered (manual): %v", gl)
			return true
		}
	}
	return false
}

func main() {
	var gameLines []*GameLine

	for _, path := range os.Args[1:] {
		data, err := ioutil.ReadFile(path)
		basename := filepath.Base(path)
		Fatal(err)
		cur := 0
		for i, v := range data {
			if v == 0 {
				if cur != i {
					shiftjis := data[cur:i]
					line := parseJIS(shiftjis)
					if line != "" {
						gl := &GameLine{File: basename, Offset: i, Length: len(shiftjis), OriginalText: line}
						if !manuallyFiltered(gl) {
							gameLines = append(gameLines, gl)
						}
						// fmt.Printf("%s,%d,%d,%q\n", basename, i, len(shiftjis), line)
					}
				}
				cur = i + 1
			}
		}
	}
	csv, err := gocsv.MarshalString(&gameLines)
	Fatal(err)
	fmt.Print(csv)
}
