package dump

import "github.com/davecgh/go-spew/spew"

// Struct выводит структуру в подробном виде
func Struct(s interface{}) string {
	return spew.Sdump(s)
}
