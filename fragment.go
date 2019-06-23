package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gocarina/gocsv"
	"golang.org/x/text/encoding/japanese"
)

var (
	outputFolder = flag.String("outputFolder", "", "output folder")
)

func Fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var jisDecoder = japanese.ShiftJIS.NewDecoder()

// \xFF5F-\xFF9F is half width kana and punctuation  (e.g. ｟ ｠ ｡ ｢ ｣ ､ ･ ｦ ｧ ｨ ｩ )
// \xFF01-\xFF5E is full width alphanumeric (e.g. ０ １ ２ ３)
// \x3000-\x303F japanese punctionation
const CHARS = "[\\p{Han}\\p{Hiragana}\\p{Katakana}" +
	"\n\x20-\x7E・ー“ΔΘ…○×ΣΛΩ※―”↑★△㎏￥∑↓" +
	"\\x{2E80}-\\x{2FD5}\\x{FF5F}-\\x{FF9F}\\x{FF01}-\\x{FF5E}\\x{3000}-\\x{303F}\\x{31F0}-\\x{31FF}\\x{3220}-\\x{3243}\\x{3280}-\\x{337F}]"

var re = regexp.MustCompile("^" + CHARS + "+$")
var reFilter = regexp.MustCompile(CHARS)
var jpnChar = regexp.MustCompile("[\\p{Han}\\p{Hiragana}\\p{Katakana}]")

//
func parseJIS(data []byte) string {
	utf8Bytes, err := jisDecoder.Bytes(data)
	if err != nil || len(utf8Bytes) < 2 { // Didn't parse as shift-JIS.
		return ""
	}

	return string(utf8Bytes)
}

func lengthFilter(gl *GameLine) bool {
	return gl.Length < 4
}

func asciiFilter(gl *GameLine) bool {
	for _, b := range gl.OriginalText {
		if b > 0x7f {
			return false
		}
	}
	return true
}

func validCharFilter(gl *GameLine) bool {
	return !re.MatchString(gl.OriginalText)
}

func jpnCharFilter(gl *GameLine) bool {
	return !jpnChar.MatchString(gl.OriginalText)
}

type GameLine struct {
	File           string `csv:"FILE"`
	Offset         int    `csv:"OFFSET"`
	Length         int    `csv:"LENGTH"`
	OriginalText   string `csv:"ORIGINAL_TEXT"`
	TranslatedText string `csv:"TRANSLATED_TEXT"`
	Status         string `csv:"STATUS"`
	TlLength       int    `csv:"TL_LENGTH"`
	Notes          string `csv:"NOTES"`
	TextKey        string `csv:"TEXT_KEY"`
}

type ManualFilter struct {
	File        string
	StartOffset int // inclusive
	EndOffset   int // inclusive
}

var manualFilters []*ManualFilter = []*ManualFilter{
	{"DEMOT.PRG", 3793, 30630},
	{"MATCHING.PRG", 1114, 1561520},
	{"GCMNO.PRG", 253, 1062160},
	{"DESKTOPF.PRG", 537, 175726},
	{"GCMNF.PRG", 810, 1582672},
	{"GCMNF.PRG", 1717132, 1722056},
	{"GCMNF.PRG", 1593760, 1616784},
	{"GCMNF.PRG", 1624192, 1624376},
	{"GCMNO.PRG", 1109024, 1129996},
	{"DEMOT.PRG", 1114, 3414},
	{"GCMNF.PRG", 1821456, 1863840},
	{"GCMNF.PRG", 1864432, 1865841},
	{"DEMOT.PRG", 33520, 34136},
	{"TOPPAGEF.PRG", 9, 204612},
	{"TOPPAGEF.PRG", 240888, 242268},
	{"MATCHING.PRG", 1633836, 1639760},
}

func manuallyFiltered(gl *GameLine) bool {
	for _, f := range manualFilters {
		if f.File == gl.File && gl.Offset >= f.StartOffset && gl.Offset <= f.EndOffset {
			// log.Printf("filtered (manual): %v", gl)
			return true
		}
	}
	return false
}

func combine(orig []*GameLine) []*GameLine {
	var combined []*GameLine

	skip := 0
	for i, gl := range orig {
		if skip > 0 {
			skip--
			continue
		}
		for j := i + 1; j < len(orig); j++ {
			cont := orig[j]
			if gl.Offset+gl.Length+1 == cont.Offset {
				// log.Printf("%v seems to be continuation of %v", gl, cont)
				gl.OriginalText += "\n" + cont.OriginalText
				gl.Length += cont.Length + 1
				skip++
			}
		}
		combined = append(combined, gl)
	}
	return combined
}

func textKey(text string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
}

type TLLine struct {
	TextKey        string `csv:"TEXT_KEY"`
	Length         int    `csv:"LENGTH"`
	TlLength       int    `csv:"TL_LENGTH"`
	OriginalText   string `csv:"ORIGINAL_TEXT"`
	TranslatedText string `csv:"TRANSLATED_TEXT"`
	Notes          string `csv:"NOTES"`
	Status         string `csv:"STATUS"`
	TlCredit       string `csv:"TL_CREDIT"`
	OrigLines      int    `csv:"ORIG_LINES"`
	TlLines        int    `csv:"TL_LINES"`
	LineStatus     string `csv:"LINE_STATUS"`
}

func uniqueTLLines(orig []*GameLine) []*TLLine {
	tls := []*TLLine{}

	lineMap := map[string]bool{}
	for _, gl := range orig {
		if !lineMap[gl.OriginalText] {
			tls = append(tls, &TLLine{Length: gl.Length, OriginalText: gl.OriginalText, TextKey: gl.TextKey})
			lineMap[gl.OriginalText] = true
		}
	}
	return tls
}

func main() {
	flag.Parse()
	log.Print(os.Args)
	var gameLines []*GameLine

	for _, path := range os.Args[2:] {
		data, err := ioutil.ReadFile(path)

		// data = bytes.Replace(data, []byte{0, '#'}, []byte{'\n', '#'}, -1)

		basename := filepath.Base(path)
		Fatal(err)
		cur := 0
		for i, v := range data {
			if v == 0 {
				if cur != i {
					shiftjis := data[cur:i]
					line := parseJIS(shiftjis)
					if line != "" {
						gl := &GameLine{File: basename, Offset: cur, Length: len(shiftjis), OriginalText: line}
						if !validCharFilter(gl) && !manuallyFiltered(gl) {
							gameLines = append(gameLines, gl)
						}
						// fmt.Printf("%s,%d,%d,%q\n", basename, i, len(shiftjis), line)
					}
				}
				cur = i + 1
			}
		}
	}
	gameLines = combine(gameLines)
	// filter(gameLines)

	for _, gl := range gameLines {
		gl.TextKey = textKey(gl.OriginalText)
	}

	var filteredGameLines []*GameLine

	for _, gl := range gameLines {
		if lengthFilter(gl) ||
			asciiFilter(gl) ||
			validCharFilter(gl) ||
			manuallyFiltered(gl) ||
			jpnCharFilter(gl) {
			continue
		}
		filteredGameLines = append(filteredGameLines, gl)
	}
	gameLines = filteredGameLines

	gameLinesCsv, err := gocsv.MarshalBytes(gameLines)
	Fatal(err)
	err = ioutil.WriteFile(filepath.Join(*outputFolder, "gamelines.csv"), gameLinesCsv, 0644)
	Fatal(err)

	tlLines := uniqueTLLines(gameLines)
	tlLinesCsv, err := gocsv.MarshalBytes(tlLines)
	Fatal(err)
	err = ioutil.WriteFile(filepath.Join(*outputFolder, "tllines.csv"), tlLinesCsv, 0644)
	Fatal(err)

}
