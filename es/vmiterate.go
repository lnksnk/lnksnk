package es

import (
	"iter"
	"reflect"
	"runtime"
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

				next, stop := iter.Pull(rslt[0].Seq())
				if next != nil && stop != nil {
					var strctseq = &SeqIter{next: next, stop: stop}
					runtime.SetFinalizer(strctseq, func(strctseqf *SeqIter) {
						go func() {
							runtime.SetFinalizer(strctseqf, nil)
							strctseqf.next = nil
							strctseqf.Value = nil
							stop := strctseqf.stop
							strctseqf.stop = nil
							if stop != nil {
								stop()
							}
						}()
					})

					iter := func() *iteratorRecord {
						iter := vm.r.toObject(vm.r.ToValue(strctseq))

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
			}
		}
	}
	panic(vm.r.NewTypeError("object is not iterable"))
}

type SeqIter struct {
	Value interface{}
	Done  bool
	next  func() (reflect.Value, bool)
	stop  func()
}

var doneseqiter = &SeqIter{Value: nil, Done: true}

func (seqitr *SeqIter) Stop() {
	if seqitr == nil {
		return
	}
	if stop := seqitr.stop; stop != nil {
		runtime.SetFinalizer(seqitr, nil)
		seqitr.next = nil
		seqitr.stop = nil
		stop()
	}
}

func (seqitr *SeqIter) Next() *SeqIter {
	if seqitr == nil {
		return doneseqiter
	}
	if next := seqitr.next; next != nil {
		rfltvl, rflcvk := next()
		if rflcvk {
			seqitr.Value = rfltvl.Interface()
			seqitr.Done = !rflcvk
			return seqitr
		}
	}
	seqitr.Stop()
	return doneseqiter
}
