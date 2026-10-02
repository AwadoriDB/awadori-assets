package rich

import "fmt"

func paint(code string, tag string) string {
	return "\x1b[" + code + "m" + tag + "\x1b[0m"
}

func Info(text string, a ...any) {
	fmt.Println(paint("34", ">>> [Info]"), fmt.Sprintf(text, a...))
}

func Error(text string, a ...any) {
	fmt.Println(paint("31", ">>> [Error]"), fmt.Sprintf(text, a...))
}

func Warning(text string, a ...any) {
	fmt.Println(paint("33", ">>> [Warning]"), fmt.Sprintf(text, a...))
}

func Panic(text string, a ...any) {
	Error(text, a...)
	panic("Exiting program due to the aforementioned reasons.")
}

func PanicError(text string, err error, a ...any) {
	Error(text, a...)
	panic(err)
}
