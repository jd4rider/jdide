package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	//"github.com/alecthomas/chroma/v2/quick"
	"github.com/mattn/go-runewidth"
	//"github.com/nsf/termbox-go"
	"github.com/gdamore/tcell/termbox"
)

var mode int

var ROWS, COLS int
var offsetCol, offsetRow int
var currentCol, currentRow int

var source_file string

var line_number_width = 0
var highlight = 1
var text_buffer = [][]rune{}
var undo_buffer = [][]rune{}
var copy_buffer = []rune{}
var modified bool

func read_file(filename string) {
	file, err := os.Open(filename)

	if err != nil {
		source_file = filename
		text_buffer = append(text_buffer, []rune{})
		return
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		line := scanner.Text()
		text_buffer = append(text_buffer, []rune{})

		for i := 0; i < len(line); i++ {
			text_buffer[lineNumber] = append(text_buffer[lineNumber], rune(line[i]))
		}
		lineNumber++
	}
	if lineNumber == 0 {
		text_buffer = append(text_buffer, []rune{})
	}
}

func write_file(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for row, line := range text_buffer {
		new_line := "\n"
		if row == len(text_buffer)-1 {
			new_line = ""
		}
		write_line := string(line) + new_line
		_, err = writer.WriteString(write_line)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		writer.Flush()
		modified = false
	}
}

func insert_rune(event termbox.Event) {
	insert_rune := make([]rune, len(text_buffer[currentRow])+1)
	copy(insert_rune[:currentCol], text_buffer[currentRow][:currentCol])
	if event.Key == termbox.KeySpace {
		insert_rune[currentCol] = rune(' ')
	} else if event.Key == termbox.KeyTab {
		insert_rune[currentCol] = rune('\t')
	} else {
		insert_rune[currentCol] = rune(event.Ch)
	}
	copy(insert_rune[currentCol+1:], text_buffer[currentRow][currentCol:])
	text_buffer[currentRow] = insert_rune
	currentCol++
}

func delete_rune() {
	if currentCol > 0 {
		currentCol--
		delete_line := make([]rune, len(text_buffer[currentRow])-1)
		copy(delete_line[:currentCol], text_buffer[currentRow][:currentCol])
		copy(delete_line[currentCol:], text_buffer[currentRow][currentCol+1:])
		text_buffer[currentRow] = delete_line
	} else if currentRow > 0 {
		append_line := make([]rune, len(text_buffer[currentRow]))
		copy(append_line, text_buffer[currentRow][currentCol:])
		new_text_buffer := make([][]rune, len(text_buffer)-1)
		copy(new_text_buffer[:currentRow], text_buffer[:currentRow])
		copy(new_text_buffer[currentRow:], text_buffer[currentRow+1:])
		text_buffer = new_text_buffer
		currentRow--
		currentCol = len(text_buffer[currentRow])
		insert_line := make([]rune, len(text_buffer[currentRow])+len(append_line))
		copy(insert_line[:len(text_buffer[currentRow])], text_buffer[currentRow])
		copy(insert_line[len(text_buffer[currentRow]):], append_line)
		text_buffer[currentRow] = insert_line
	}
}

func insert_line() {
	right_line := make([]rune, len(text_buffer[currentRow][currentCol:]))
	copy(right_line, text_buffer[currentRow][currentCol:])
	left_line := make([]rune, len(text_buffer[currentRow][:currentCol]))
	copy(left_line, text_buffer[currentRow][:currentCol])
	text_buffer[currentRow] = left_line
	currentRow++
	currentCol = 0
	new_text_buffer := make([][]rune, len(text_buffer)+1)
	copy(new_text_buffer, text_buffer[:currentRow])
	new_text_buffer[currentRow] = right_line
	copy(new_text_buffer[currentRow+1:], text_buffer[currentRow:])
	text_buffer = new_text_buffer
}

func copy_line() {
	copy_line := make([]rune, len(text_buffer[currentRow]))
	copy(copy_line, text_buffer[currentRow])
	copy_buffer = copy_line
}

func cut_line() {
	copy_line()
	if currentRow == len(text_buffer) || len(text_buffer) < 2 {
		return
	}
	new_text_buffer := make([][]rune, len(text_buffer)-1)
	copy(new_text_buffer[:currentRow], text_buffer[:currentRow])
	copy(new_text_buffer[currentRow:], text_buffer[currentRow+1:])
	text_buffer = new_text_buffer
	if currentRow > 0 {
		currentRow--
		currentCol = 0
	}
}

func paste_line() {
	if len(copy_buffer) == 0 {
		currentRow++
		currentCol = 0
	}
	new_text_buffer := make([][]rune, len(text_buffer)+1)
	copy(new_text_buffer[:currentRow], text_buffer[:currentRow])
	new_text_buffer[currentRow] = copy_buffer
	copy(new_text_buffer[currentRow+1:], text_buffer[currentRow:])
	text_buffer = new_text_buffer
}

func push_buffer() {
	copy_undo_buffer := make([][]rune, len(text_buffer))
	copy(copy_undo_buffer, text_buffer)
	undo_buffer = copy_undo_buffer
}

func pull_buffer() {
	if len(undo_buffer) == 0 {
		return
	}
	text_buffer = undo_buffer
}

func scroll_text_buffer() {
	if currentRow < offsetRow {
		offsetRow = currentRow
	}
	if currentCol < offsetCol {
		offsetCol = currentCol
	}
	if currentRow >= offsetRow+ROWS {
		offsetRow = currentRow - ROWS + 1
	}
	if currentCol >= offsetCol+COLS {
		offsetCol = currentCol - COLS + 1
	}
}

func highlight_keyword(keyword string, col, row int) {
	for i := 0; i < len(keyword); i++ {
		ch := text_buffer[row+offsetRow][col+offsetCol+i]
		termbox.SetCell(col+i+line_number_width, row, ch, termbox.ColorGreen|termbox.AttrBold, termbox.ColorDefault)
	}
}

func highlight_string(col, row int) int {
	i := 0
	for {
		if col+offsetCol+i == len(text_buffer[row+offsetRow]) {
			return i - 1
		}
		ch := text_buffer[row+offsetRow][col+offsetCol+i]
		if ch == '"' || ch == '\'' {
			termbox.SetCell(col+i+line_number_width, row, ch, termbox.ColorYellow, termbox.ColorDefault)
			return i
		} else {
			termbox.SetCell(col+i+line_number_width, row, ch, termbox.ColorYellow, termbox.ColorDefault)
			i++
		}
	}
}

func highlight_comment(col, row int) int {
	i := 0
	for {
		if col+offsetCol+i == len(text_buffer[row+offsetRow]) {
			return i - 1
		}
		ch := text_buffer[row+offsetRow][col+offsetCol+i]
		termbox.SetCell(col+i+line_number_width, row, ch, termbox.ColorMagenta|termbox.AttrBold, termbox.ColorDefault)
		i++
	}
}

func highlight_syntax(col *int, row, text_buffer_col, text_buffer_row int) {
	ch := text_buffer[text_buffer_row][text_buffer_col]
	next_token := string(text_buffer[text_buffer_row][text_buffer_col:])
	if unicode.IsDigit(ch) {
		termbox.SetCell(*col+line_number_width, row, ch, termbox.ColorYellow|termbox.AttrBold, termbox.ColorDefault)
	} else if ch == '\'' {
		termbox.SetCell(*col+line_number_width, row, ch, termbox.ColorYellow, termbox.ColorDefault)
		*col++
		*col += highlight_string(*col, row)
	} else if ch == '"' {
		termbox.SetCell(*col+line_number_width, row, ch, termbox.ColorYellow, termbox.ColorDefault)
		*col++
		*col += highlight_string(*col, row)
	} else if strings.Contains("+-*><=%&|^!:", string(ch)) {
		termbox.SetCell(*col+line_number_width, row, ch, termbox.ColorMagenta|termbox.AttrBold, termbox.ColorDefault)
	} else if ch == '/' {
		termbox.SetCell(*col+line_number_width, row, ch, termbox.ColorMagenta|termbox.AttrBold, termbox.ColorDefault)
		index := strings.Index(next_token, "//")
		if index == 0 {
			*col += highlight_comment(*col, row)
		}
		index = strings.Index(next_token, "/*")
		if index == 0 {
			*col += highlight_comment(*col, row)
		}
	} else if ch == '#' {
		termbox.SetCell(*col+line_number_width, row, ch, termbox.ColorMagenta|termbox.AttrBold, termbox.ColorDefault)
		*col += highlight_comment(*col, row)
	} else {
		for _, token := range []string{
			"false", "False", "NaN", "None", "bool", "break", "byte",
			"case", "catch", "class", "const", "continue", "def", "do",
			"elif", "else", "else:", "enum", "export", "extends", "extern",
			"finally", "float", "for", "from", "func", "function",
			"global", "if", "import", " in", "int", "lambda", "try:", "except:",
			"nil", "not", "null", "pass", "print", "raise", "return",
			"self", "short", "signed", "sizeof", "static", "struct", "switch",
			"this", "throw", "throws", "true", "True", "typedef", "typeof",
			"undefined", "union", "unsigned", "until", "var", "void",
			"while", "with", "yield", "double", "select", "where", "and", "order", "having",
		} {
			index := strings.Index(next_token, token+" ")
			if index == 0 {
				highlight_keyword(token[:len(token)], *col, row)
				*col += len(token)
				break
			} else {
				termbox.SetCell(*col+line_number_width, row, ch, termbox.ColorDefault, termbox.ColorDefault)
			}
		}
	}
}

func display_text_buffer() {
	var row, col int
	for row = 0; row < ROWS; row++ {
		text_bufferRow := row + offsetRow
		for col = 0; col < COLS; col++ {
			text_bufferCol := col + offsetCol
			if text_bufferRow >= 0 && text_bufferRow < len(text_buffer) && text_bufferCol < len(text_buffer[text_bufferRow]) {
				if text_buffer[text_bufferRow][text_bufferCol] != '\t' {
					if highlight == 1 {
						highlight_syntax(&col, row, text_bufferCol, text_bufferRow)
					} else {
						termbox.SetCell(col, row, text_buffer[text_bufferRow][text_bufferCol], termbox.ColorDefault, termbox.ColorDefault)
					}
				} else {
					termbox.SetCell(col, row, rune(' '), termbox.ColorDefault, termbox.ColorDefault)
				}
				//err := quick.Highlight(os.Stdout, string(text_buffer), "go", "terminal256", "gruvbox")
				//if err != nil {
				//	fmt.Println(err)
				//	os.Exit(1)
				//}
			} else if row+offsetRow > len(text_buffer)-1 {
				termbox.SetCell(0, row, rune('*'), termbox.ColorBlue, termbox.ColorDefault)
				termbox.SetCell(col, row, rune('\n'), termbox.ColorDefault, termbox.ColorDefault)
			}
		}
	}
}

func display_status_bar() {
	var mode_status string
	var file_status string
	var copy_status string
	var undo_status string
	var cursor_status string

	if mode > 0 {
		mode_status = " EDIT: "
	} else {
		mode_status = " VIEW: "
	}

	filename_length := len(source_file)
	if filename_length > 8 {
		filename_length = 8
	}

	file_status = source_file[:filename_length] + " - " + strconv.Itoa(len(text_buffer)) + " lines"
	if modified {
		file_status += " modified"
	} else {
		file_status += " saved"
	}
	cursor_status = " Row " + strconv.Itoa(currentRow+1) + ", Col " + strconv.Itoa(currentCol+1) + " "
	if len(copy_buffer) > 0 {
		copy_status = " [Copy] "
	}
	if len(undo_buffer) > 0 {
		undo_status = " [Undo] "
	}
	used_space := len(mode_status) + len(file_status) + len(cursor_status) + len(copy_status) + len(undo_status)
	spaces := strings.Repeat(" ", COLS-used_space)
	message := mode_status + file_status + copy_status + undo_status + spaces + cursor_status
	print_message(0, ROWS, termbox.ColorBlack, termbox.ColorWhite, message)
}

func print_message(col, row int, fg, bg termbox.Attribute, message string) {
	for _, ch := range message {
		termbox.SetCell(col, row, ch, fg, bg)
		col += runewidth.RuneWidth(ch)
	}
}

func get_key() termbox.Event {
	var key_event termbox.Event
	switch event := termbox.PollEvent(); event.Type {
	case termbox.EventKey:
		key_event = event
	case termbox.EventError:
		panic(event.Err)
	}
	return key_event
}

func process_keypress() {
	key_event := get_key()
	if key_event.Key == termbox.KeyEsc {
		mode = 0
	} else if key_event.Ch != 0 {
		if mode == 1 {
			insert_rune(key_event)
			modified = true
		} else {
			switch key_event.Ch {
			case 'q':
				termbox.Close()
				os.Exit(0)
			case 'i':
				mode = 1
			case 'w':
				write_file(source_file)
			case 'y':
				copy_line()
			case 'p':
				paste_line()
			case 'd':
				cut_line()
			case 's':
				push_buffer()
			case 'u':
				pull_buffer()
			case 'k':
				if currentRow != 0 {
					currentRow--
				}
			case 'j':
				if currentRow < len(text_buffer)-1 {
					currentRow++
				}
			case 'h':
				if currentCol != 0 {
					currentCol--
				} else if currentRow > 0 {
					currentRow--
					currentCol = len(text_buffer[currentRow])
				}
			case 'l':
				if currentCol < len(text_buffer[currentRow]) {
					currentCol++
				} else if currentRow < len(text_buffer)-1 {
					currentRow++
					currentCol = 0
				}
			case 'z':
				if highlight == 1 {
					highlight = 0
				} else {
					highlight = 1
				}
			}
		}
	} else {
		switch key_event.Key {
		case termbox.KeyEnter:
			if mode == 1 {
				insert_line()
				modified = true
			}
		case termbox.KeyBackspace:
			delete_rune()
			modified = true
		case termbox.KeyBackspace2:
			delete_rune()
			modified = true
		case termbox.KeyTab:
			if mode == 1 {
				insert_rune(key_event)
				modified = true
			}
		case termbox.KeySpace:
			if mode == 1 {
				insert_rune(key_event)
				modified = true
			}
		case termbox.KeyHome:
			currentCol = 0
		case termbox.KeyEnd:
			currentCol = len(text_buffer[currentCol])
		case termbox.KeyPgup:
			if currentRow-int(ROWS/4) > 0 {
				currentRow -= int(ROWS / 4)
			}
		case termbox.KeyPgdn:
			if currentRow+int(ROWS/4) < len(text_buffer)-1 {
				currentRow += int(ROWS / 4)
			}
		case termbox.KeyArrowUp:
			if currentRow != 0 {
				currentRow--
			}
		case termbox.KeyArrowDown:
			if currentRow < len(text_buffer)-1 {
				currentRow++
			}
		case termbox.KeyArrowLeft:
			if currentCol != 0 {
				currentCol--
			} else if currentRow > 0 {
				currentRow--
				currentCol = len(text_buffer[currentRow])
			}
		case termbox.KeyArrowRight:
			if currentCol < len(text_buffer[currentRow]) {
				currentCol++
			} else if currentRow < len(text_buffer)-1 {
				currentRow++
				currentCol = 0
			}
		}
		switch key_event.Ch {
		}
		if currentCol > len(text_buffer[currentRow]) {
			currentCol = len(text_buffer[currentRow])
		}
	}
}

func run_editor() {
	err := termbox.Init()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		source_file = os.Args[1]
		read_file(source_file)
	} else {
		source_file = "out.txt"
		text_buffer = append(text_buffer, []rune{})
	}

	//currentRow = 36

	for {
		COLS, ROWS = termbox.Size()
		ROWS--
		if COLS < 78 {
			COLS = 78
		}
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		scroll_text_buffer()
		display_text_buffer()
		display_status_bar()
		termbox.SetCursor(currentCol-offsetCol, currentRow-offsetRow)
		termbox.Flush()
		process_keypress()

	}
}

func main() {
	run_editor()
}
