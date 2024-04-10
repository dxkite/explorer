package stream

import (
	"io"

	"dxkite.cn/explore-me/src/core/binary"
)

type JsonStream struct {
	r io.ReadSeeker
}

func NewJsonStream(r io.ReadSeekCloser) *JsonStream {
	return &JsonStream{r: r}
}

func (j *JsonStream) Offset(offset int64) error {
	_, err := j.r.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

func (j *JsonStream) ScanNext(rst interface{}) (int64, interface{}, error) {
	cur, err := j.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, nil, err
	}
	if err := binary.Read(j.r, rst); err != nil {
		return cur, nil, err
	}
	return cur, rst, nil
}
