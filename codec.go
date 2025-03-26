package testexec

import (
	"encoding/gob"
	"errors"
	"io"
	"os"
	"reflect"
)

func discard(r io.Reader) error {
	_, err := io.Copy(io.Discard, r)
	if errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
}

func decodeValues(v reflect.Value, r io.Reader) error {
	if v.IsNil() {
		return discard(r)
	}

	dec := gob.NewDecoder(r)
	el := reflect.New(v.Type().Elem())
	defer v.Close()
	for {
		if err := dec.DecodeValue(el); err == nil {
			v.Send(el.Elem())
		} else if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
			return nil
		} else {
			return err
		}
	}
}

func decodeSingle[T any](v *T, r io.Reader) error {
	dec := gob.NewDecoder(r)
	if err := dec.Decode(v); err != nil {
		return err
	}

	var tmp struct{}
	if err := dec.Decode(&tmp); errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
		return nil
	}
	return errors.New("too many values to decode, expect EOF")
}

func encodeValues(v reflect.Value, w io.WriteCloser) error {
	if v.IsNil() {
		return w.Close()
	}

	enc := gob.NewEncoder(w)
	for {
		el, ok := v.Recv()
		if !ok {
			return w.Close()
		}

		if err := enc.EncodeValue(el); err != nil {
			_ = w.Close()
			return err
		}
	}
}

func encodeSingle[T any](v T, w io.WriteCloser) error {
	if err := gob.NewEncoder(w).Encode(v); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}
