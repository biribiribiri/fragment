package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/biribiribiri/iso9660"
	"github.com/gocarina/gocsv"
	"golang.org/x/text/encoding/japanese"
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
	translatedCsv = flag.String("translatedCsv", "https://docs.google.com/spreadsheets/d/e/2PACX-1vTEeAsyZAWYOq3NouuFryyl-U7Tk0Xxe0rar1ZPrM_qUou170P8-BX5fS2w-9tPLnzq6VquXWj3qrpO/pub?gid=1884693932&single=true&output=csv", "path to translated csv")
	isoPath       = flag.String("isoPath", "", "Path to original fragment ISO")
	outputIsoPath = flag.String("outputIsoPath", "", "outputIsoPath")
	jisEncoder    = japanese.ShiftJIS.NewEncoder()
)

func filesInIsoDirectory(isoFile *iso9660.File, base string, extentMap map[string]iso9660.Extent) {
	children, err := isoFile.GetChildren()
	if err != nil {
		log.Fatalf("failed to get iso folder children: %s", err)
	}
	var sep string
	if base != "" {
		sep = "/"
	}
	for _, child := range children {
		if child.IsDir() {
			filesInIsoDirectory(child, base+sep+child.Name(), extentMap)
		} else {
			extentMap[base+sep+child.Name()] = child.Extent()
		}
	}
}

func filesInIso(ra io.ReaderAt) map[string]iso9660.Extent {
	img, err := iso9660.OpenImage(ra)
	if err != nil {
		log.Fatalf("failed to open iso: %s", err)
	}
	root, err := img.RootDir()
	if err != nil {
		log.Fatalf("failed to open iso root dir: %s", err)
	}
	m := make(map[string]iso9660.Extent)
	filesInIsoDirectory(root, "", m)
	return m
}

func downloadCsv() []byte {
	resp, err := http.Get(*translatedCsv)
	if err != nil {
		log.Fatalf("failed to download csv from google sheets: %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body
}

func main() {
	flag.Parse()
	fmt.Println("fragment patcher by plm")

	f, err := os.Open(*isoPath)
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}
	defer f.Close()

	extentMap := filesInIso(f)
	log.Printf("%s", extentMap)

	newIsoData, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("failed to read input iso: %s", err)
	}

	var gameLines []*GameLine
	gameLinesMap := map[string][]*GameLine{}

	data := downloadCsv()
	Fatal(gocsv.UnmarshalBytes(data, &gameLines))

	fmt.Printf("found %v game lines\n", len(gameLines))
	for _, line := range gameLines {
		if line.TranslatedText == "" {
			continue
		}
		jis, err := jisEncoder.String(line.TranslatedText)
		Fatal(err)
		if len(jis) > line.Length {
			log.Panic("line %v is too long", line)
			continue
		}
		line.File = "DATA/" + line.File
		line.TranslatedText = jis
		gameLinesMap[line.File] = append(gameLinesMap[line.File], line)
	}
	fmt.Print(gameLinesMap)
	for filename, lines := range gameLinesMap {
		extent, ok := extentMap[filename]
		if !ok {
			log.Fatalf("could not file %q in iso extent map", filename)
		}
		fileData := newIsoData[extent.Start : extent.Start+extent.Length]

		log.Printf("processing file %v", filename)
		for _, line := range lines {
			log.Print("processing line: ", line)
			for i := 0; i < line.Length+1; i++ {
				if i < len(line.TranslatedText) {
					// log.Printf("replacing offset %v, %v with %v", line.Offset+i, fileData[line.Offset+i], line.TranslatedText[i])
					if line.TranslatedText[i] == '\n' {
						fileData[line.Offset+i] = 0
					} else {
						fileData[line.Offset+i] = line.TranslatedText[i]
					}
				} else if i < line.Length {
					// log.Printf("replacing offset %v, %v with 0", line.Offset+i, fileData[line.Offset+i])
					// fileData[line.Offset+i] = ' '
					fileData[line.Offset+i] = 0
				} else {
					fileData[line.Offset+i] = 0
				}
			}
		}
	}
	Fatal(ioutil.WriteFile(*outputIsoPath, newIsoData, 0644))
}
