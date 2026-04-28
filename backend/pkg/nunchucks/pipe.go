package nunchucks

import (
	"encoding/json"
	"strings"
)

type Step func(value any, args ...any) any

type PipelineStep struct {
	Fn   Step
	Args []any
}

type Pipe struct {
	Value any
}

func S(fn Step) PipelineStep {
	return PipelineStep{Fn: fn}
}

func SA(fn Step, args ...any) PipelineStep {
	return PipelineStep{
		Fn:   fn,
		Args: args,
	}
}

func PipeOf(value any) Pipe {
	return Pipe{Value: value}
}

func (p Pipe) To(cb Step, args ...any) Pipe {
	return PipeOf(cb(p.Value, args...))
}

func Combiner(steps []PipelineStep) func(x any) any {
	return func(x any) any {
		acc := PipeOf(x)
		for _, step := range steps {
			if step.Fn == nil {
				continue
			}

			if len(step.Args) > 0 {
				parsed := make([]any, len(step.Args))
				for i, a := range step.Args {
					parsed[i] = ParseVar(a)
				}

				acc = acc.To(step.Fn, parsed...)
			} else {
				acc = acc.To(step.Fn)
			}
		}
		return acc.Value
	}
}

func ParseVar(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case bool:
		return x
	case int, int64, float64:
		return x
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return ""
		}
		if lit, ok := parseLiteral(s); ok {
			return lit
		}
		if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
			var out any
			if err := json.Unmarshal([]byte(s), &out); err == nil {
				return out
			}
		}
		return x
	default:
		return v
	}
}
