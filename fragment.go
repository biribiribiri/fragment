package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

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
	"\x20-\x7F・ー“ΔΘ…○×ΣΛΩ※―”↑★△㎏￥∑↓" +
	"\\x{2E80}-\\x{2FD5}\\x{FF5F}-\\x{FF9F}\\x{FF01}-\\x{FF5E}\\x{3000}-\\x{303F}\\x{31F0}-\\x{31FF}\\x{3220}-\\x{3243}\\x{3280}-\\x{337F}]"

var re = regexp.MustCompile("^" + CHARS + "+$")
var reFilter = regexp.MustCompile(CHARS)

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
		invalid := reFilter.ReplaceAllString(string(utf8Bytes), "")
		log.Printf("filtered (regex): %s [%v]", utf8Bytes, invalid)
		return ""
	}

	// if len(re.FindAll(utf8Bytes, 4)) < 4 {
	// 	return ""
	// }

	return string(utf8Bytes)
}

func main() {
	fmt.Printf("FILE,OFFSET,LEN,TEXT\n")
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
						fmt.Printf("%s,%d,%d,%s\n", basename, i, len(shiftjis), line)
					}
				}
				cur = i + 1
			}
		}
	}
}
