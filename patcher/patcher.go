package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/gocarina/gocsv"
)

type GameLine struct {
	File           string `csv:"FILE"`
	Offset         int    `csv:"OFFSET"`
	Length         int    `csv:"LENGTH"`
	OriginalText   string `csv:"ORIGINAL_TEXT"`
	TranslatedText string `csv:"TRANSLATED_TEXT"`
	Status         string `csv:"STATUS"`
	TlLength       int    `csv:"TL_LENGTH"`
	Notes          string `csv:"NOTES"`
}

func Fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var (
	translatedCsv = flag.String("translatedCsv", "", "path to translated csv")
	inputFolder   = flag.String("inputFolder", "", "original game folder")
	outputFolder  = flag.String("outputFolder", "", "patched output")
)

func main() {
	flag.Parse()
	fmt.Println("fragment patcher by plm")

	var gameLines []*GameLine
	gameLinesMap := map[string][]*GameLine{}

	data, err := ioutil.ReadFile(*translatedCsv)
	Fatal(err)
	Fatal(gocsv.UnmarshalBytes(data, &gameLines))

	fmt.Printf("found %v game lines\n", len(gameLines))
	for _, line := range gameLines {
		if line.TranslatedText == "" {
			continue
		}
		if len(line.TranslatedText) > line.Length {
			log.Panic("line %v is too long", line)
			continue
		}
		line.File = "DATA/" + line.File
		gameLinesMap[line.File] = append(gameLinesMap[line.File], line)
	}
	fmt.Print(gameLinesMap)
	for filename, lines := range gameLinesMap {
		path := filepath.Join(*inputFolder, filename)
		outPath := filepath.Join(*outputFolder, filename)
		log.Printf("processing input %v to output %v", path, outPath)
		fileData, err := ioutil.ReadFile(path)
		Fatal(err)
		for _, line := range lines {
			log.Print("processing line: ", line)
			for i := 0; i < line.Length+1; i++ {
				if i < len(line.TranslatedText) {
					// TODO: convert TranslatedText to shiftjis
					// log.Printf("replacing offset %v, %v with %v", line.Offset+i, fileData[line.Offset+i], line.TranslatedText[i])
					if line.TranslatedText[i] == '\n' {
						fileData[line.Offset+i] = 0
					} else {
						fileData[line.Offset+i] = line.TranslatedText[i]
					}
				} else if i < line.Length {
					// log.Printf("replacing offset %v, %v with 0", line.Offset+i, fileData[line.Offset+i])
					fileData[line.Offset+i] = ' '
				} else {
					fileData[line.Offset+i] = 0
				}
			}
		}
		Fatal(ioutil.WriteFile(outPath, fileData, 0644))
	}
}
