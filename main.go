package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/clayts/docscii/docBook"

	"github.com/fatih/color"
)

func readArgs() (string, string, Style) {
	flag.Usage = func() {
		fmt.Println(color.GreenString("docscii") + " v2\nDocBook to AsciiDoc converter by Clayton Spicer\n\nUsage:\n  docscii input_dir output_dir\n  Or:\n  docscii input/publican.cfg output_dir\n  Or:\n  db2d input_file.xml output_dir\n\nOptions:")
		flag.PrintDefaults()
	}
	var input, output string
	s := NewStyle()
	var cQuotes, mQuotes, iQuotes, sQuotes, bQuotes, hQuotes string
	flag.StringVar(&cQuotes, "custom", strings.Join(DefaultStyle["custom"], ","), "comma-separated list of custom semantic tags to preserve")
	flag.StringVar(&mQuotes, "monospace", strings.Join(DefaultStyle["monospace"], ","), "comma-separated list of DocBook elements to render as in-line literal text")
	flag.StringVar(&sQuotes, "superscript", strings.Join(DefaultStyle["superscript"], ","), "comma-separated list of DocBook elements to render as in-line superscript text")
	flag.StringVar(&iQuotes, "italic", strings.Join(DefaultStyle["italic"], ","), "comma-separated list of DocBook elements to render as in-line italic text")
	flag.StringVar(&bQuotes, "bold", strings.Join(DefaultStyle["bold"], ","), "comma-separated list of DocBook elements to render as in-line bold text")
	flag.StringVar(&hQuotes, "highlight", strings.Join(DefaultStyle["highlight"], ","), "comma-separated list of DocBook elements to render as in-line highlighted text")

	flag.Parse()
	s.AddFromString("custom", cQuotes)
	s.AddFromString("monospace", mQuotes)
	s.AddFromString("superscript", sQuotes)
	s.AddFromString("italics", iQuotes)
	s.AddFromString("bold", bQuotes)
	s.AddFromString("highlight", hQuotes)
	if len(flag.Args()) < 2 {
		flag.Usage()
		os.Exit(1)
	} else {
		input = flag.Arg(0)
		output = flag.Arg(1)
	}
	return input, output, s
}

func main() {
	input, output, s := readArgs()
	log.Println("Converting", color.CyanString(input), "to", color.CyanString(output))
	db := docBook.NewFromPublicanCfg(input)
	if db == nil {
		db = docBook.NewFromDir(input)
	}
	if db == nil {
		db = docBook.NewFromFile(input)
	}
	if db == nil {
		panic("file not found")
	}
	fmt.Print("Processing...\t")
	ad := AsciiDocFromDocBook(db, s)
	fmt.Println(" Complete.")
	ad.Write(output)
	masterfile, _ := filepath.Abs(output + "/master.adoc")
	log.Println("Complete:", color.CyanString(masterfile))
}
