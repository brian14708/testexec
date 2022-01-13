package testexec

import (
	"encoding/gob"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"reflect"
)

func decodeValues(v reflect.Value, r io.Reader) error {
	if !v.IsValid() {
		// discard value
		_, err := io.Copy(ioutil.Discard, r)
		if errors.Is(err, os.ErrClosed) {
			return nil
		}
		return err
	}

	if v.Type().Kind() != reflect.Chan {
		dec := gob.NewDecoder(r)
		if err := dec.DecodeValue(v); err != nil {
			return err
		}

		var tmp struct{}
		if err := dec.Decode(&tmp); errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
			return nil
		}
		return errors.New("too many values to decode, expect EOF")
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

func encodeValues(v reflect.Value, w io.WriteCloser) error {
	if !v.IsValid() {
		return w.Close()
	}

	if v.Type().Kind() != reflect.Chan {
		if err := gob.NewEncoder(w).EncodeValue(v); err != nil {
			w.Close()
			return err
		}
		return w.Close()
	}

	enc := gob.NewEncoder(w)
	for {
		el, ok := v.Recv()
		if !ok {
			return w.Close()
		}

		if err := enc.EncodeValue(el); err != nil {
			w.Close()
			return err
		}
	}
}
