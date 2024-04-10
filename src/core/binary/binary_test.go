package binary

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"testing"
)

type EfwRecord struct {
	ID            int64   `json:"a"`
	IdempotenceId string  `json:"b"`
	Val           []int   `json:"c"`
	Var           [10]int `json:"v"`
	EfwRecordA
	V *EfwRecordA `json:"c"`
}

type EfwRecordA struct {
	ID            int64  `json:"e"`
	IdempotenceId string `json:"f"`
}

func TestRead(t *testing.T) {
	buf := &bytes.Buffer{}
	e := &EfwRecord{}
	e.EfwRecordA.ID = 1
	e.EfwRecordA.IdempotenceId = "test"
	e.ID = 2
	e.IdempotenceId = "test20"
	e.Val = []int{1, 2, 3}
	e.Var = [10]int{1, 2, 3, 4, 5}

	if err := Write(buf, e); err != nil {
		t.Error(err)
		return
	}

	vv, _ := json.Marshal(e)
	fmt.Println(buf.Len(), hex.EncodeToString(buf.Bytes()))
	fmt.Println(len(vv), string(vv))
	fmt.Println(strconv.Quote(buf.String()))

	e1 := &EfwRecord{}
	if err := Read(buf, e1); err != nil {
		t.Error(err)
		return
	}

	fmt.Println(e, e1)

	if !reflect.DeepEqual(e, e1) {
		t.Error("not equal")
	}
}

func Test_readInt(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := writeInt(buf, 100); err != nil {
		t.Error(err)
		return
	}

	buf.Write([]byte{0, 0, 0, 0})

	fmt.Println("all", hex.EncodeToString(buf.Bytes()))

	var rd int64
	err := readInt(buf, &rd)
	if err != nil {
		t.Error("readInt", err)
		return
	}

	least, err := io.ReadAll(buf)
	if err != nil {
		t.Error("io.ReadAll", err)
		return
	}

	fmt.Println("least", hex.EncodeToString(least))
}
