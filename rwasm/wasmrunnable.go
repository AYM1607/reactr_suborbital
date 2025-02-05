package rwasm

import (
	"encoding/json"

	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm/moduleref"
	"github.com/suborbital/reactr/rwasm/runtime"

	"github.com/pkg/errors"
)

//Runner represents a wasm-based runnable
type Runner struct {
	env *runtime.WasmEnvironment
}

// NewRunner returns a new *Runner
func NewRunner(filepath string) *Runner {
	ref := &moduleref.WasmModuleRef{
		Filepath: filepath,
	}

	return NewRunnerWithRef(ref)
}

func NewRunnerWithRef(ref *moduleref.WasmModuleRef) *Runner {
	builder := runtimeBuilder(ref)

	environment := runtime.NewEnvironment(builder)

	r := &Runner{
		env: environment,
	}

	return r
}

// Run runs a Runner
func (w *Runner) Run(job rt.Job, ctx *rt.Ctx) (interface{}, error) {
	var jobBytes []byte
	var req *request.CoordinatedRequest

	// check if the job is a CoordinatedRequest (pointer or bytes), and set up the WasmInstance if so
	if jobReq, ok := job.Data().(*request.CoordinatedRequest); ok {
		req = jobReq

	} else if jobReq, err := request.FromJSON(job.Bytes()); err == nil {
		req = jobReq

	} else {
		// if it's not a request, treat it as normal data
		bytes, bytesErr := interfaceToBytes(job.Data())
		if bytesErr != nil {
			return nil, errors.Wrap(bytesErr, "failed to parse job for Wasm Runnable")
		}

		jobBytes = bytes
	}

	if req != nil {
		ctx.UseRequest(req)
		jobBytes = req.Body
	}

	var output []byte
	var runErr error
	var callErr error

	if err := w.env.UseInstance(ctx, func(instance *runtime.WasmInstance, ident int32) {
		inPointer, writeErr := instance.WriteMemory(jobBytes)
		if writeErr != nil {
			runErr = errors.Wrap(writeErr, "failed to instance.writeMemory")
			return
		}

		// execute the Runnable's Run function, passing the input data and ident
		// set runErr but don't return because the ExecutionResult error should also be grabbed
		_, callErr = instance.Call("run_e", inPointer, int32(len(jobBytes)), ident)

		// get the results from the instance
		output, runErr = instance.ExecutionResult()

		// deallocate the memory used for the input
		instance.Deallocate(inPointer, len(jobBytes))
	}); err != nil {
		return nil, errors.Wrap(err, "failed to useInstance")
	}

	if runErr != nil {
		// we do not wrap the error here as we want to
		// propogate its exact type to the caller (specifically rt.RunErr)
		return nil, runErr
	}

	if callErr != nil {
		// if the runnable didn't return an explicit runErr, still check to see if there was an
		// error executing the module in the first place. It's posslble for both to be non-nil
		// in which case returning the runErr takes precedence, which is why it's checked first.
		return nil, errors.Wrap(callErr, "wasm execution error")
	}

	if req != nil {
		resp := &request.CoordinatedResponse{
			Output:      output,
			RespHeaders: req.RespHeaders,
		}

		return resp, nil
	}

	return output, nil
}

// OnChange runs when a worker starts using this Runnable
func (w *Runner) OnChange(evt rt.ChangeEvent) error {
	switch evt {
	case rt.ChangeTypeStart:
		if err := w.env.AddInstance(); err != nil {
			return errors.Wrap(err, "failed to addInstance")
		}
	case rt.ChangeTypeStop:
		if err := w.env.RemoveInstance(); err != nil {
			return errors.Wrap(err, "failed to removeInstance")
		}
	}

	return nil
}

func interfaceToBytes(data interface{}) ([]byte, error) {
	// if data is []byte or string, return it as-is
	if b, ok := data.([]byte); ok {
		return b, nil
	} else if s, ok := data.(string); ok {
		return []byte(s), nil
	}

	// otherwise, assume it's a struct of some kind,
	// so JSON marshal it and return it
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal job data")
	}

	return dataJSON, nil
}
