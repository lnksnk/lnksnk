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
	seq    iter.Seq[reflect.Value]
	Value  interface{}
	Done   bool
	nxtval chan interface{}
	nxtrd  chan bool
	fin    func()
}

func (itrsq *_iterseq) Close() (err error) {
	if itrsq == nil {
		return
	}
	fin := itrsq.fin
	itrsq.fin = nil
	itrsq.Value = nil
	itrsq.Done = false
	nxtrd := itrsq.nxtrd
	itrsq.nxtrd = nil
	nxtval := itrsq.nxtval
	itrsq.nxtval = nil
	if fin != nil {
		fin()
	}
	if nxtrd != nil {
		close(nxtrd)
	}
	if nxtval != nil {
		close(nxtval)
	}
	return
}

func (itrsq *_iterseq) Next() *_iterseq {
	if itrsq == nil {
		return itrsq
	}
	if seq := itrsq.seq; seq != nil {
		itrsq.seq = nil
		itrsq.nxtrd = make(chan bool, 1)
		itrsq.nxtval = make(chan interface{}, 1)
		go func(v chan interface{}, nxt chan bool) {
			defer func() {

			}()
			for rflctv := range seq {
				v <- rflctv.Interface()
				itrsq.Done = false
				if <-nxt {
					continue
				}
				return
			}
			itrsq.Done = true
			v <- nil
		}(itrsq.nxtval, itrsq.nxtrd)
		itrsq.Value = <-itrsq.nxtval
	} else {
		itrsq.nxtrd <- true
		itrsq.Value = <-itrsq.nxtval
	}
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
	itrsq.nxtrd <- false
	itrsq.Done = true
	itrsq.Value = nil
	fin := itrsq.fin
	itrsq.fin = nil
	if fin != nil {
		fin()
	}
	close(itrsq.nxtrd)
	itrsq.nxtrd = nil
	close(itrsq.nxtval)
	itrsq.nxtval = nil
	return itrsq
}
