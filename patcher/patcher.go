package main

import (
	"flag"
	"io/ioutil"
	"log"

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
	outputFolder  = flag.String("inputFolder", "", "original game folder")
)

func main() {
	flag.Parse()

	var gameLines []*GameLine
	data, err := ioutil.ReadFile(*translatedCsv)
	Fatal(err)
	Fatal(gocsv.UnmarshalBytes(data, &gameLines))

	for _, line := range gameLines {
		if line.TranslatedText == "" {
			continue
		}
		if len(line.TranslatedText) > line.Length {
			log.Errorf("line %v is too long", line)
			continue
		}
	}
}
