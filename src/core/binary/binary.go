package binary

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
)

func Write(w io.Writer, v interface{}) error {
	return write(w, reflect.Indirect(reflect.ValueOf(v)))
}

func write(w io.Writer, v reflect.Value) error {
	debug("write", v.Type().String())

	switch v.Kind() {
	case reflect.Struct:
		l := v.NumField()

		if err := writeInt(w, int64(l)); err != nil {
			return err
		}

		for i := 0; i < l; i++ {
			if err := write(w, v.Field(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Slice:
		l := v.Len()
		if err := writeInt(w, int64(l)); err != nil {
			return err
		}

		for i := 0; i < l; i++ {
			if err := write(w, v.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Array:
		l := v.Len()
		for i := 0; i < l; i++ {
			if err := write(w, v.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.String:
		l := v.Len()
		if err := writeInt(w, int64(l)); err != nil {
			return err
		}

		d := v.String()
		_, err := w.Write([]byte(d))
		if err != nil {
			return err
		}
		return nil

	case reflect.Int:
		return writeInt(w, v.Int())

	case reflect.Pointer:
		l := 1
		if v.IsNil() {
			l = 0
		}
		if l == 0 {
			if err := writeInt(w, int64(l)); err != nil {
				return err
			}
			return nil
		}

		return write(w, v.Elem())
	}

	return binary.Write(w, binary.BigEndian, v.Interface())
}

func writeInt(w io.Writer, v int64) error {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(buf, int64(v))
	if _, err := w.Write(buf[:n]); err != nil {
		return err
	}
	return nil
}

func readInt(r io.Reader, v *int64) error {
	vv, err := binary.ReadVarint(byteReader{r: r})
	if err != nil {
		return err
	}
	*v = vv
	return nil
}

type byteReader struct {
	r io.Reader
}

func (r byteReader) ReadByte() (byte, error) {
	buf := make([]byte, 1)
	if n, err := r.r.Read(buf); err != nil {
		return 0, err
	} else if n != 1 {
		return 0, io.EOF
	} else {
		return buf[0], nil
	}
}

func Read(r io.Reader, v interface{}) error {
	return read(r, reflect.Indirect(reflect.ValueOf(v)))
}

func read(r io.Reader, v reflect.Value) error {
	debug("read", v.Type().String())

	switch v.Kind() {
	case reflect.Struct:
		var l int64

		if err := readInt(r, &l); err != nil {
			return err
		}

		or := getSortFields(v.Type())

		if int64(len(or)) < l {
			return errors.New("error fields size")
		}

		for _, i := range or[:l] {
			if err := read(r, v.Field(i)); err != nil {
				return err
			}
		}

		return nil
	case reflect.Slice:
		var l int64

		if err := readInt(r, &l); err != nil {
			return err
		}

		arr := reflect.MakeSlice(v.Type(), int(l), int(l))
		for i := 0; int64(i) < l; i++ {
			if err := read(r, arr.Index(i)); err != nil {
				return err
			}
		}
		v.Set(arr)
		return nil

	case reflect.Array:
		l := v.Len()
		for i := 0; i < l; i++ {
			if err := read(r, v.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.String:
		var l int64

		if err := readInt(r, &l); err != nil {
			return err
		}

		buf := make([]byte, l)
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}

		v.SetString(string(buf))
		return nil

	case reflect.Int:
		var rv int64
		if err := readInt(r, &rv); err != nil {
			return err
		}
		v.SetInt(rv)
		return nil
	case reflect.Pointer:
		var l int64
		if err := readInt(r, &l); err != nil {
			return err
		}

		if l == 0 {
			return nil
		}

		return read(r, v.Elem())
	}

	return binary.Read(r, binary.BigEndian, v.Addr().Interface())
}

type field struct {
	j int
	i int
}

func getSortFields(t reflect.Type) []int {
	l := t.NumField()
	fields := []*field{}

	for i := 0; i < l; i++ {
		ii := t.Field(i).Tag.Get("index")
		bi, _ := strconv.ParseUint(ii, 10, 64)
		idx := i
		if bi > 0 {
			idx = int(bi)
		}
		fields = append(fields, &field{
			j: i,
			i: idx,
		})
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].i < fields[j].i
	})

	arr := []int{}
	for _, v := range fields {
		arr = append(arr, v.j)
	}
	return arr
}

var debugIt = os.Getenv("DEBUG")

func debug(msg ...interface{}) {
	if debugIt == "true" {
		fmt.Println(msg...)
	}
}
