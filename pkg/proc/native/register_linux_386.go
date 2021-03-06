package native

import (
	"fmt"

	sys "golang.org/x/sys/unix"

	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/pkg/proc/amd64util"
	"github.com/go-delve/delve/pkg/proc/linutil"
)

// SetPC sets EIP to the value specified by 'pc'.
func (thread *nativeThread) SetPC(pc uint64) error {
	ir, err := registers(thread)
	if err != nil {
		return err
	}
	r := ir.(*linutil.I386Registers)
	r.Regs.Eip = int32(pc)
	thread.dbp.execPtraceFunc(func() { err = sys.PtraceSetRegs(thread.ID, (*sys.PtraceRegs)(r.Regs)) })
	return err
}

// SetSP sets ESP to the value specified by 'sp'
func (thread *nativeThread) SetSP(sp uint64) (err error) {
	var ir proc.Registers
	ir, err = registers(thread)
	if err != nil {
		return err
	}
	r := ir.(*linutil.I386Registers)
	r.Regs.Esp = int32(sp)
	thread.dbp.execPtraceFunc(func() { err = sys.PtraceSetRegs(thread.ID, (*sys.PtraceRegs)(r.Regs)) })
	return
}

func (thread *nativeThread) SetDX(dx uint64) (err error) {
	var ir proc.Registers
	ir, err = registers(thread)
	if err != nil {
		return err
	}
	r := ir.(*linutil.I386Registers)
	r.Regs.Edx = int32(dx)
	thread.dbp.execPtraceFunc(func() { err = sys.PtraceSetRegs(thread.ID, (*sys.PtraceRegs)(r.Regs)) })
	return
}

func registers(thread *nativeThread) (proc.Registers, error) {
	var (
		regs linutil.I386PtraceRegs
		err  error
	)
	thread.dbp.execPtraceFunc(func() { err = sys.PtraceGetRegs(thread.ID, (*sys.PtraceRegs)(&regs)) })
	if err != nil {
		return nil, err
	}
	r := linutil.NewI386Registers(&regs, func(r *linutil.I386Registers) error {
		var fpregset amd64util.AMD64Xstate
		var floatLoadError error
		r.Fpregs, fpregset, floatLoadError = thread.fpRegisters()
		r.Fpregset = &fpregset
		return floatLoadError
	})
	thread.dbp.execPtraceFunc(func() {
		tls, _ := ptraceGetTls(regs.Xgs, thread.ThreadID())
		r.Tls = uint64(tls)
	})
	return r, nil
}

const _NT_X86_XSTATE = 0x202

func (thread *nativeThread) fpRegisters() (regs []proc.Register, fpregs amd64util.AMD64Xstate, err error) {
	thread.dbp.execPtraceFunc(func() { fpregs, err = ptraceGetRegset(thread.ID) })
	regs = fpregs.Decode()
	if err != nil {
		err = fmt.Errorf("could not get floating point registers: %v", err.Error())
	}
	return
}
