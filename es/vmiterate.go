package es

import (
	"iter"
	"reflect"
)

type _iterateGen struct{}

var iterateGen _iterateGen

func (_iterateGen) exec(vm *vm) {
	obj := vm.stack[vm.sp-1]
	method := toMethod(vm.r.getV(obj, SymIterator))
	if method != nil {
		iter := func() *iteratorRecord {
			iter := vm.r.toObject(method(FunctionCall{
				This: obj,
			}))

			var next func(FunctionCall) Value

			if obj, ok := iter.self.getStr("next", nil).(*Object); ok {
				if call, ok := obj.self.assertCallable(); ok {
					next = call
				}
			}

			return &iteratorRecord{
				iterator: iter,
				next:     next,
			}
		}()
		vm.iterStack = append(vm.iterStack, iterStackItem{iter: iter})
		vm.sp--
		vm.pc++
		return
	}
	if psblitr := obj.Export(); psblitr != nil {
		var itertpe = reflect.TypeOf(psblitr)
		if itertpe.Kind() == reflect.Func && itertpe.NumOut() == 1 && itertpe.Out(0).CanSeq() {
			if rslt := reflect.ValueOf(psblitr).Call(nil); len(rslt) > 0 {
				var done = false
				var itrsq *_iterseq
				defer func() {
					if itrsq != nil {
						itrsq.Close()
					}
				}()
				iter := func() *iteratorRecord {
					itrsq = &_iterseq{seq: rslt[0].Seq(), fin: func() {
						done = true
					}}
					iter := vm.r.toObject(vm.r.ToValue(itrsq))

					var next func(FunctionCall) Value

					if obj, ok := iter.self.getStr("next", nil).(*Object); ok {
						if call, ok := obj.self.assertCallable(); ok {
							next = call
						}
					}

					return &iteratorRecord{
						iterator: iter,
						next:     next}
				}()

				vm.iterStack = append(vm.iterStack, iterStackItem{iter: iter})
				vm.sp--
				vm.pc++
				for vmexec(vm, vm.pc) {
					if done {
						itrsq.Close()
						itrsq = nil
						return
					}
				}
				return
			}
		}
	}
	panic(vm.r.NewTypeError("object is not iterable"))
}

type _iterseq struct {
	itrnext func() (interface{}, bool)
	itrstop func()
	seq     iter.Seq[reflect.Value]
	Value   interface{}
	Done    bool
	fin     func()
}

func (itrsq *_iterseq) Close() (err error) {
	if itrsq == nil {
		return
	}
	fin := itrsq.fin
	itrsq.fin = nil
	itrsq.Value = nil
	itrsq.Done = false
	itrstop := itrsq.itrstop
	itrsq.itrstop = nil
	itrsq.itrnext = nil
	if fin != nil {
		fin()
	}
	if itrstop != nil {
		itrstop()
	}
	return
}

func (itrsq *_iterseq) Next() *_iterseq {
	if itrsq == nil {
		return itrsq
	}
	if seq := itrsq.seq; seq != nil {
		itrnext, itrstop := iter.Pull(seq)
		itrsq.itrnext = func() (val interface{}, vld bool) {
			val, vld = itrnext()
			if vld {
				vld = !vld
				val = val.(reflect.Value).Interface()
				return
			}
			val = nil
			vld = true
			return
		}
		itrsq.itrstop = itrstop
		itrsq.seq = nil
	}
	if itrnext := itrsq.itrnext; itrnext != nil {
		itrsq.Value, itrsq.Done = itrnext()
		if fin := itrsq.fin; fin != nil && itrsq.Done {
			itrsq.fin = nil
			fin()
		}
		return itrsq
	}
	itrsq.Value, itrsq.Done = nil, true
	if fin := itrsq.fin; fin != nil && itrsq.Done {
		itrsq.fin = nil
		fin()
	}
	return itrsq
}

func (itrsq *_iterseq) Return() *_iterseq {
	if itrsq == nil {
		return itrsq
	}
	itrsq.Value = nil
	itrsq.Done = true
	itrsq.itrnext = nil
	if itrstop := itrsq.itrstop; itrstop != nil {
		itrsq.itrstop = nil
		itrstop()
	}
	fin := itrsq.fin
	itrsq.fin = nil
	if fin != nil {
		fin()
	}
	return itrsq
}
