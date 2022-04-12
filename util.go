package testexec

import "reflect"

func typeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func setValue[T any](v *T, u reflect.Value) {
	reflect.ValueOf(v).Elem().Set(u)
}

func makeChan[T any](s int) reflect.Value {
	return reflect.MakeChan(
		reflect.ChanOf(reflect.BothDir, typeOf[T]().Elem()), s,
	)
}
