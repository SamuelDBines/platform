package nunchucks

import "strings"

// ConfigOptions controls environment setup for rendering templates.
type ConfigOptions struct {
	Path                string
	Loader              Loader
	VariableStart       string
	VariableEnd         string
	BlockStart          string
	BlockEnd            string
	Globals             map[string]any
	GlobalTemplates     []string
	GlobalHeadTemplates []string
	GlobalFootTemplates []string
}

// Env is the Go renderer environment.
type Env struct {
	basePath            string
	loader              Loader
	variableStart       string
	variableEnd         string
	blockStart          string
	blockEnd            string
	globals             map[string]any
	globalTemplates     []string
	globalHeadTemplates []string
	globalFootTemplates []string
}

const (
	defaultVariableStart = "{{"
	defaultVariableEnd   = "}}"
	defaultBlockStart    = "{%"
	defaultBlockEnd      = "%}"
)

func builtinGlobals() map[string]any {
	return map[string]any{
		"range": TemplateFunc(func(args []any, _ map[string]any, _ string) (any, error) {
			start := 0
			stop := 0
			step := 1

			switch len(args) {
			case 1:
				stop = toInt(args[0], 0)
			case 2:
				start = toInt(args[0], 0)
				stop = toInt(args[1], 0)
			default:
				if len(args) >= 3 {
					start = toInt(args[0], 0)
					stop = toInt(args[1], 0)
					step = toInt(args[2], 1)
				}
			}

			if step == 0 {
				step = 1
			}

			out := []any{}
			if step > 0 {
				for i := start; i < stop; i += step {
					out = append(out, i)
				}
				return out, nil
			}
			for i := start; i > stop; i += step {
				out = append(out, i)
			}
			return out, nil
		}),
	}
}

// Configure builds a rendering environment using options similar to the TS API.
func Configure(opts ConfigOptions) *Env {
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		path = "views"
	}

	variableStart := strings.TrimSpace(opts.VariableStart)
	if variableStart == "" {
		variableStart = defaultVariableStart
	}
	variableEnd := strings.TrimSpace(opts.VariableEnd)
	if variableEnd == "" {
		variableEnd = defaultVariableEnd
	}
	blockStart := strings.TrimSpace(opts.BlockStart)
	if blockStart == "" {
		blockStart = defaultBlockStart
	}
	blockEnd := strings.TrimSpace(opts.BlockEnd)
	if blockEnd == "" {
		blockEnd = defaultBlockEnd
	}

	ldr := opts.Loader
	if ldr == nil {
		ldr = FileSystemLoader(path)
	}

	globals := make([]string, 0, len(opts.GlobalTemplates))
	for _, name := range opts.GlobalTemplates {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		globals = append(globals, n)
	}

	headGlobals := make([]string, 0, len(opts.GlobalHeadTemplates))
	for _, name := range opts.GlobalHeadTemplates {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		headGlobals = append(headGlobals, n)
	}

	footGlobals := make([]string, 0, len(opts.GlobalFootTemplates))
	for _, name := range opts.GlobalFootTemplates {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		footGlobals = append(footGlobals, n)
	}

	configuredGlobals := cloneMap(opts.Globals)
	if configuredGlobals == nil {
		configuredGlobals = map[string]any{}
	}

	return &Env{
		basePath:            path,
		loader:              ldr,
		variableStart:       variableStart,
		variableEnd:         variableEnd,
		blockStart:          blockStart,
		blockEnd:            blockEnd,
		globals:             configuredGlobals,
		globalTemplates:     globals,
		globalHeadTemplates: headGlobals,
		globalFootTemplates: footGlobals,
	}
}

// Render loads and renders a template file with the provided context.
func (e *Env) Render(name string, ctx map[string]any) (string, error) {
	raw, err := e.readRawTemplate(name)
	if err != nil {
		return "", err
	}
	renderCtx := e.buildRenderContext(ctx)
	if err := ApplyTemplateContractDefaults(e.normalizeTemplateSource(raw), renderCtx); err != nil {
		return "", err
	}
	if err := ValidateTemplateContract(e.normalizeTemplateSource(raw), renderCtx); err != nil {
		return "", err
	}
	src, err := e.compileTemplate(name)
	if err != nil {
		return "", err
	}
	return e.renderString(src, renderCtx)
}

// Compile resolves includes/extends into a compiled template string.
func (e *Env) Compile(name string) (string, error) {
	return e.compileTemplate(name)
}

// RenderString renders a string template with the provided context.
func (e *Env) RenderString(src string, ctx map[string]any) (string, error) {
	return e.renderString(src, ctx)
}

func (e *Env) normalizeTemplateSource(src string) string {
	replacerArgs := []string{}
	if e.variableStart != defaultVariableStart {
		replacerArgs = append(replacerArgs, e.variableStart, defaultVariableStart)
	}
	if e.variableEnd != defaultVariableEnd {
		replacerArgs = append(replacerArgs, e.variableEnd, defaultVariableEnd)
	}
	if e.blockStart != defaultBlockStart {
		replacerArgs = append(replacerArgs, e.blockStart, defaultBlockStart)
	}
	if e.blockEnd != defaultBlockEnd {
		replacerArgs = append(replacerArgs, e.blockEnd, defaultBlockEnd)
	}
	if len(replacerArgs) == 0 {
		return src
	}
	return strings.NewReplacer(replacerArgs...).Replace(src)
}
