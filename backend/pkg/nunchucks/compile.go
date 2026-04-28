package nunchucks

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var includeRe = regexp.MustCompile(`\{%\s*include\s+(["'][^"']+["'])\s*%\}`)
var extendsRe = regexp.MustCompile(`\{%\s*extends\s+(["'][^"']+["'])\s*%\}`)
var compileStmtRe = regexp.MustCompile(`\{%\s*([A-Za-z_][A-Za-z0-9_]*)\b([^%]*)%\}`)

func unquote(s string) string {
	t := strings.TrimSpace(s)
	if len(t) >= 2 {
		if (t[0] == '"' && t[len(t)-1] == '"') || (t[0] == '\'' && t[len(t)-1] == '\'') {
			return t[1 : len(t)-1]
		}
	}
	return t
}

func (e *Env) readRawTemplate(name string) (string, error) {
	res := e.loader.Read(name)
	if res.Err != "" {
		return "", fmt.Errorf(res.Err)
	}
	return e.normalizeTemplateSource(res.Res), nil
}

func (e *Env) readTemplate(name string) (string, error) {
	raw, err := e.readRawTemplate(name)
	if err != nil {
		return "", err
	}
	return stripComments(raw), nil
}

func (e *Env) resolveIncludes(src string, seen map[string]bool) (string, error) {
	out := src
	for {
		m := includeRe.FindStringSubmatchIndex(out)
		if m == nil {
			return out, nil
		}

		tok := out[m[2]:m[3]]
		name := unquote(tok)
		if seen[name] {
			out = out[:m[0]] + out[m[1]:]
			continue
		}

		child, err := e.readTemplate(name)
		if err != nil {
			return "", err
		}

		resolved, err := e.resolveIncludes(child, cloneSeen(seen, name))
		if err != nil {
			return "", err
		}

		out = out[:m[0]] + resolved + out[m[1]:]
	}
}

func cloneSeen(src map[string]bool, next string) map[string]bool {
	out := make(map[string]bool, len(src)+1)
	for k, v := range src {
		out[k] = v
	}
	out[next] = true
	return out
}

type blockSpan struct {
	name      string
	openStart int
	openEnd   int
	bodyStart int
	bodyEnd   int
	closeEnd  int
}

func extractBlocks(src string) map[string]blockSpan {
	blocks := map[string]blockSpan{}
	type open struct {
		name      string
		openStart int
		openEnd   int
	}
	stack := []open{}

	tagRe := regexp.MustCompile(`\{%\s*([A-Za-z_][A-Za-z0-9_]*)\b([^%]*)%\}`)
	all := tagRe.FindAllStringSubmatchIndex(src, -1)
	for _, m := range all {
		kw := strings.TrimSpace(src[m[2]:m[3]])
		rest := strings.TrimSpace(src[m[4]:m[5]])
		start := m[0]
		end := m[1]

		if kw == "block" {
			name := strings.Fields(rest)
			if len(name) == 0 {
				continue
			}
			stack = append(stack, open{name: name[0], openStart: start, openEnd: end})
			continue
		}

		if kw == "endblock" {
			if len(stack) == 0 {
				continue
			}
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			blocks[last.name] = blockSpan{
				name:      last.name,
				openStart: last.openStart,
				openEnd:   last.openEnd,
				bodyStart: last.openEnd,
				bodyEnd:   start,
				closeEnd:  end,
			}
		}
	}

	return blocks
}

func mergeExtends(baseSrc, childSrc string) string {
	baseBlocks := extractBlocks(baseSrc)
	childBlocks := extractBlocks(childSrc)

	type edit struct {
		start int
		end   int
		val   string
	}
	edits := []edit{}

	for name, bb := range baseBlocks {
		cb, ok := childBlocks[name]
		if !ok {
			continue
		}

		baseBody := baseSrc[bb.bodyStart:bb.bodyEnd]
		childBody := childSrc[cb.bodyStart:cb.bodyEnd]
		childBody = strings.ReplaceAll(childBody, "{{ super() }}", baseBody)
		edits = append(edits, edit{start: bb.bodyStart, end: bb.bodyEnd, val: childBody})
	}

	out := baseSrc
	sort.Slice(edits, func(i, j int) bool {
		return edits[i].start > edits[j].start
	})
	for _, e := range edits {
		out = out[:e.start] + e.val + out[e.end:]
	}
	return out
}

func removeExtendsTag(src string) string {
	return extendsRe.ReplaceAllString(src, "")
}

func extractExtendsPrelude(src string) string {
	type frame struct{ kind string }
	var out strings.Builder
	stack := []frame{}
	tags := compileStmtRe.FindAllStringSubmatchIndex(src, -1)

	for _, m := range tags {
		kw := strings.TrimSpace(src[m[2]:m[3]])
		raw := src[m[0]:m[1]]

		switch kw {
		case "block", "if", "for", "macro", "call", "filter", "raw", "verbatim", "client":
			stack = append(stack, frame{kind: kw})
			continue
		case "endblock", "endif", "endfor", "endmacro", "endcall", "endfilter", "endraw", "endverbatim", "endclient":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		case "extends":
			continue
		case "set":
			if len(stack) == 0 {
				out.WriteString(raw)
				if !strings.HasSuffix(raw, "\n") {
					out.WriteString("\n")
				}
			}
		}
	}

	return out.String()
}

func (e *Env) compileTemplate(name string) (string, error) {
	entry, err := e.readTemplate(name)
	if err != nil {
		return "", err
	}

	child := entry

	seen := map[string]bool{name: true}
	for {
		m := extendsRe.FindStringSubmatch(child)
		if m == nil {
			return removeExtendsTag(child), nil
		}

		baseName := unquote(m[1])
		if seen[baseName] {
			return "", fmt.Errorf("extends cycle detected")
		}
		seen[baseName] = true

		baseRaw, err := e.readTemplate(baseName)
		if err != nil {
			return "", err
		}
		base := baseRaw

		prelude := extractExtendsPrelude(child)
		child = prelude + mergeExtends(base, child)
		child = removeExtendsTag(child)
	}
}
