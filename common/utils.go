package common

import (
	"fmt"
	"github.com/grafana/sobek"
)

// ToBytes tries to return a byte slice from compatible types.
func ToBytes(data interface{}) ([]byte, error) {
	switch dt := data.(type) {
	case []byte:
		return dt, nil
	case string:
		return []byte(dt), nil
	case sobek.ArrayBuffer:
		return dt.Bytes(), nil
	default:
		return nil, fmt.Errorf("invalid type %T, expected string, []byte or ArrayBuffer", data)
	}
}

func Throw(rt *sobek.Runtime, err error) {
	if e, ok := err.(*sobek.Exception); ok { //nolint:errorlint // we don't really want to unwrap here
		panic(e)
	}
	panic(rt.NewGoError(err)) // this catches the stack unlike rt.ToValue
}
