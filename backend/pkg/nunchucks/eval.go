package nunchucks

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"math/rand"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TemplateFunc func(args []any, kwargs map[string]any, caller string) (any, error)

type missingValue struct{}

func (missingValue) String() string { return "" }

var missing = missingValue{}

var callableExprRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\s*\((.*)\)$`)
var stripTagsRe = regexp.MustCompile(`(?s)<[^>]*>`)
var urlizeRe = regexp.MustCompile(`https?://[^\s<]+`)
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func getPath(data map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var cur any = data
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, exists := m[p]
		if !exists {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func parseLiteral(s string) (any, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", true
	}
	if (strings.HasPrefix(t, `"`) && strings.HasSuffix(t, `"`)) || (strings.HasPrefix(t, "'") && strings.HasSuffix(t, "'")) {
		return t[1 : len(t)-1], true
	}
	switch t {
	case "true":
		return true, true
	case "false":
		return false, true
	case "null", "nil":
		return nil, true
	}
	if i, err := strconv.Atoi(t); err == nil {
		return i, true
	}
	if f, err := strconv.ParseFloat(t, 64); err == nil {
		return f, true
	}
	return nil, false
}

func splitArgs(src string) []string {
	out := []string{}
	cur := strings.Builder{}
	depth := 0
	quote := byte(0)
	esc := false

	for i := 0; i < len(src); i++ {
		ch := src[i]
		if esc {
			cur.WriteByte(ch)
			esc = false
			continue
		}
		if quote != 0 {
			if ch == '\\' {
				esc = true
				cur.WriteByte(ch)
				continue
			}
			cur.WriteByte(ch)
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			cur.WriteByte(ch)
			continue
		}
		switch ch {
		case '(', '[', '{':
			depth++
			cur.WriteByte(ch)
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
			cur.WriteByte(ch)
		case ',':
			if depth == 0 {
				t := strings.TrimSpace(cur.String())
				if t != "" {
					out = append(out, t)
				}
				cur.Reset()
				continue
			}
			cur.WriteByte(ch)
		default:
			cur.WriteByte(ch)
		}
	}
	t := strings.TrimSpace(cur.String())
	if t != "" {
		out = append(out, t)
	}
	return out
}

func splitTopLevelAssign(src string) (string, string, bool) {
	s := strings.TrimSpace(src)
	depth := 0
	quote := byte(0)
	esc := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if esc {
			esc = false
			continue
		}
		if quote != 0 {
			if ch == '\\' {
				esc = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '=':
			if depth == 0 {
				left := strings.TrimSpace(s[:i])
				right := strings.TrimSpace(s[i+1:])
				if left != "" {
					return left, right, true
				}
			}
		}
	}
	return "", "", false
}

func parseCallableExpr(src string) (string, []string, bool) {
	s := strings.TrimSpace(src)
	m := callableExprRe.FindStringSubmatch(s)
	if m == nil {
		return "", nil, false
	}
	name := strings.TrimSpace(m[1])
	argsRaw := strings.TrimSpace(m[2])
	if argsRaw == "" {
		return name, nil, true
	}
	return name, splitArgs(argsRaw), true
}

func parseCallableArgs(argExprs []string) ([]string, map[string]string) {
	pos := make([]string, 0, len(argExprs))
	kw := map[string]string{}
	for _, raw := range argExprs {
		if k, v, ok := splitTopLevelAssign(raw); ok {
			kw[k] = v
			continue
		}
		pos = append(pos, raw)
	}
	return pos, kw
}

func resolveIdentEx(name string, vars, ctx map[string]any) (any, bool) {
	key := strings.TrimSpace(name)
	if key == "" {
		return missing, false
	}

	if v, ok := vars[key]; ok {
		return v, true
	}
	if v, ok := ctx[key]; ok {
		return v, true
	}

	if strings.Contains(key, ".") {
		if v, ok := getPath(vars, key); ok {
			return v, true
		}
		if v, ok := getPath(ctx, key); ok {
			return v, true
		}
	}

	return missing, false
}

func resolveIdent(name string, vars, ctx map[string]any) any {
	v, ok := resolveIdentEx(name, vars, ctx)
	if !ok {
		return missing
	}
	return v
}

func isMissing(v any) bool {
	_, ok := v.(missingValue)
	return ok
}

func toBool(v any, dflt bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		if s == "true" || s == "1" || s == "yes" {
			return true
		}
		if s == "false" || s == "0" || s == "no" {
			return false
		}
	}
	return dflt
}

func toInt(v any, dflt int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return i
		}
	}
	return dflt
}

func toFloat(v any, dflt float64) float64 {
	switch t := v.(type) {
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case float64:
		return t
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return f
		}
	}
	return dflt
}

func toSlice(v any) []any {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		return t
	case []string:
		out := make([]any, len(t))
		for i := range t {
			out[i] = t[i]
		}
		return out
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out
	}
	return nil
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	if isMissing(v) {
		return true
	}
	switch t := v.(type) {
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	case bool:
		return !t
	case int:
		return t == 0
	case float64:
		return t == 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return rv.Len() == 0
	}
	return false
}

func parseFilterSpec(raw string) (string, []string) {
	if name, args, ok := parseCallableExpr(raw); ok {
		return strings.TrimSpace(name), args
	}
	return strings.TrimSpace(raw), nil
}

func valueByPathEx(v any, path string) (any, bool) {
	if strings.TrimSpace(path) == "" {
		return v, true
	}
	parts := strings.Split(path, ".")
	cur := v
	for _, part := range parts {
		switch t := cur.(type) {
		case map[string]any:
			next, ok := t[part]
			if !ok {
				return missing, false
			}
			cur = next
		default:
			rv := reflect.ValueOf(cur)
			if !rv.IsValid() {
				return missing, false
			}
			if rv.Kind() == reflect.Map {
				mv := rv.MapIndex(reflect.ValueOf(part))
				if mv.IsValid() {
					cur = mv.Interface()
				} else {
					return missing, false
				}
			} else {
				return missing, false
			}
		}
	}
	return cur, true
}

func valueByPath(v any, path string) any {
	out, ok := valueByPathEx(v, path)
	if !ok {
		return missing
	}
	return out
}

func containsOp(container any, item any) bool {
	switch t := container.(type) {
	case string:
		return strings.Contains(t, fmt.Sprint(item))
	case []any:
		for _, v := range t {
			if equalOp(v, item) {
				return true
			}
		}
		return false
	case map[string]any:
		_, ok := t[fmt.Sprint(item)]
		return ok
	default:
		rv := reflect.ValueOf(container)
		if !rv.IsValid() {
			return false
		}
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			for i := 0; i < rv.Len(); i++ {
				if equalOp(rv.Index(i).Interface(), item) {
					return true
				}
			}
			return false
		}
	}
	return false
}

func isIterable(v any) bool {
	if isMissing(v) || v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return true
	default:
		return false
	}
}

func isNumber(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64:
		return true
	}
	return false
}

func isSequence(v any) bool {
	if isMissing(v) || v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		return true
	default:
		return false
	}
}

func isMapping(v any) bool {
	if isMissing(v) || v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Map
}

func isKnownTestName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "defined", "undefined", "none", "null", "string", "number", "boolean", "bool", "iterable", "callable", "odd", "even", "divisibleby", "lower", "upper", "equalto", "sameas", "sequence", "mapping", "true", "false":
		return true
	default:
		return false
	}
}

func evalTestKeyword(v any, kw string, args []any) bool {
	k := strings.ToLower(strings.TrimSpace(kw))
	switch k {
	case "defined":
		return !isMissing(v)
	case "undefined":
		return isMissing(v)
	case "none", "null":
		return v == nil || isMissing(v)
	case "string":
		_, ok := v.(string)
		return ok
	case "number":
		return isNumber(v)
	case "boolean", "bool":
		_, ok := v.(bool)
		return ok
	case "iterable":
		return isIterable(v)
	case "callable":
		switch v.(type) {
		case TemplateFunc, func(...any) any:
			return true
		default:
			return false
		}
	case "odd":
		return toInt(v, 0)%2 != 0
	case "even":
		return toInt(v, 0)%2 == 0
	case "divisibleby":
		if len(args) == 0 {
			return false
		}
		d := toInt(args[0], 0)
		if d == 0 {
			return false
		}
		return toInt(v, 0)%d == 0
	case "lower":
		s := fmt.Sprint(v)
		return s == strings.ToLower(s)
	case "upper":
		s := fmt.Sprint(v)
		return s == strings.ToUpper(s)
	case "equalto":
		if len(args) == 0 {
			return false
		}
		return equalOp(v, args[0])
	case "sameas":
		if len(args) == 0 {
			return false
		}
		return reflect.DeepEqual(v, args[0])
	case "sequence":
		return isSequence(v)
	case "mapping":
		return isMapping(v)
	case "true":
		return v == true
	case "false":
		return v == false
	default:
		return false
	}
}

func isTestKeyword(v any, kw string) bool {
	return evalTestKeyword(v, kw, nil)
}

func compareAny(a any, b any, caseSens bool) int {
	if isMissing(a) && isMissing(b) {
		return 0
	}
	if isMissing(a) {
		return -1
	}
	if isMissing(b) {
		return 1
	}
	if isNumber(a) && isNumber(b) {
		af := toFloat(a, 0)
		bf := toFloat(b, 0)
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0
	}
	as := fmt.Sprint(a)
	bs := fmt.Sprint(b)
	if !caseSens {
		as = strings.ToLower(as)
		bs = strings.ToLower(bs)
	}
	if as < bs {
		return -1
	}
	if as > bs {
		return 1
	}
	return 0
}

func applyFilter(name string, v any, args []any) any {
	n := strings.TrimSpace(strings.ToLower(name))
	switch n {
	case "lower":
		return strings.ToLower(fmt.Sprint(v))
	case "upper":
		return strings.ToUpper(fmt.Sprint(v))
	case "string":
		return fmt.Sprint(v)
	case "trim":
		return strings.TrimSpace(fmt.Sprint(v))
	case "title":
		return strings.Title(strings.TrimSpace(fmt.Sprint(v)))
	case "capitalize":
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
	case "abs":
		return math.Abs(toFloat(v, 0))
	case "int":
		return toInt(v, 0)
	case "float":
		return toFloat(v, 0)
	case "length":
		if arr := toSlice(v); arr != nil {
			return len(arr)
		}
		rv := reflect.ValueOf(v)
		if rv.IsValid() {
			switch rv.Kind() {
			case reflect.String, reflect.Map, reflect.Array, reflect.Slice:
				return rv.Len()
			}
		}
		return 0
	case "first":
		if arr := toSlice(v); len(arr) > 0 {
			return arr[0]
		}
		s := fmt.Sprint(v)
		if s == "" {
			return ""
		}
		return string([]rune(s)[0])
	case "last":
		if arr := toSlice(v); len(arr) > 0 {
			return arr[len(arr)-1]
		}
		s := fmt.Sprint(v)
		if s == "" {
			return ""
		}
		r := []rune(s)
		return string(r[len(r)-1])
	case "join":
		sep := ""
		if len(args) > 0 {
			sep = fmt.Sprint(args[0])
		}
		arr := toSlice(v)
		if len(arr) == 0 {
			return ""
		}
		parts := make([]string, 0, len(arr))
		for _, it := range arr {
			parts = append(parts, fmt.Sprint(it))
		}
		return strings.Join(parts, sep)
	case "list":
		if arr := toSlice(v); arr != nil {
			return arr
		}
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.Map {
			keys := rv.MapKeys()
			out := make([]any, 0, len(keys))
			for _, k := range keys {
				out = append(out, k.Interface())
			}
			sort.SliceStable(out, func(i, j int) bool {
				return compareAny(out[i], out[j], false) < 0
			})
			return out
		}
		s := fmt.Sprint(v)
		r := []rune(s)
		out := make([]any, 0, len(r))
		for _, ch := range r {
			out = append(out, string(ch))
		}
		return out
	case "replace":
		from := ""
		to := ""
		count := -1
		if len(args) > 0 {
			from = fmt.Sprint(args[0])
		}
		if len(args) > 1 {
			to = fmt.Sprint(args[1])
		}
		if len(args) > 2 {
			count = toInt(args[2], -1)
		}
		return strings.Replace(fmt.Sprint(v), from, to, count)
	case "reverse":
		if arr := toSlice(v); arr != nil {
			out := append([]any{}, arr...)
			for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
				out[i], out[j] = out[j], out[i]
			}
			return out
		}
		r := []rune(fmt.Sprint(v))
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		return string(r)
	case "round":
		prec := 0
		if len(args) > 0 {
			prec = toInt(args[0], 0)
		}
		f := toFloat(v, 0)
		p := math.Pow(10, float64(prec))
		return math.Round(f*p) / p
	case "default":
		var dflt any = ""
		if len(args) > 0 {
			dflt = args[0]
		}
		boolMode := false
		if len(args) > 1 {
			boolMode = toBool(args[1], false)
		}
		if v == nil {
			return dflt
		}
		if boolMode && isEmpty(v) {
			return dflt
		}
		if fmt.Sprint(v) == "" {
			return dflt
		}
		return v
	case "escape", "e", "forceescape":
		return html.EscapeString(fmt.Sprint(v))
	case "safe":
		return v
	case "dump":
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	case "wordcount":
		return len(strings.Fields(fmt.Sprint(v)))
	case "nl2br":
		return strings.ReplaceAll(fmt.Sprint(v), "\n", "<br />\n")
	case "urlencode":
		return url.QueryEscape(fmt.Sprint(v))
	case "urlize":
		s := fmt.Sprint(v)
		return urlizeRe.ReplaceAllStringFunc(s, func(m string) string {
			return `<a href="` + m + `">` + m + `</a>`
		})
	case "striptags":
		preserve := false
		if len(args) > 0 {
			preserve = toBool(args[0], false)
		}
		s := stripTagsRe.ReplaceAllString(fmt.Sprint(v), "")
		if preserve {
			return s
		}
		return strings.Join(strings.Fields(s), " ")
	case "truncate":
		s := fmt.Sprint(v)
		length := 255
		killwords := false
		end := "..."
		if len(args) > 0 {
			length = toInt(args[0], 255)
		}
		if len(args) > 1 {
			killwords = toBool(args[1], false)
		}
		if len(args) > 2 {
			end = fmt.Sprint(args[2])
		}
		if len([]rune(s)) <= length {
			return s
		}
		if length <= len([]rune(end)) {
			return end
		}
		cut := length - len([]rune(end))
		r := []rune(s)
		if cut < 0 {
			cut = 0
		}
		if killwords {
			return string(r[:cut]) + end
		}
		chunk := string(r[:cut])
		if idx := strings.LastIndex(chunk, " "); idx > 0 {
			chunk = chunk[:idx]
		}
		return chunk + end
	case "center":
		width := 80
		if len(args) > 0 {
			width = toInt(args[0], 80)
		}
		s := fmt.Sprint(v)
		if len(s) >= width {
			return s
		}
		pad := width - len(s)
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	case "indent":
		width := 4
		if len(args) > 0 {
			width = toInt(args[0], 4)
		}
		prefix := strings.Repeat(" ", width)
		lines := strings.Split(fmt.Sprint(v), "\n")
		for i := range lines {
			lines[i] = prefix + lines[i]
		}
		return strings.Join(lines, "\n")
	case "sum":
		arr := toSlice(v)
		total := 0.0
		for _, it := range arr {
			total += toFloat(it, 0)
		}
		return total
	case "random":
		arr := toSlice(v)
		if len(arr) == 0 {
			return ""
		}
		return arr[rnd.Intn(len(arr))]
	case "batch":
		size := 1
		if len(args) > 0 {
			size = toInt(args[0], 1)
		}
		if size <= 0 {
			size = 1
		}
		arr := toSlice(v)
		fill := any(nil)
		hasFill := len(args) > 1
		if hasFill {
			fill = args[1]
		}
		out := []any{}
		for i := 0; i < len(arr); i += size {
			end := i + size
			if end > len(arr) {
				end = len(arr)
			}
			chunk := append([]any{}, arr[i:end]...)
			if hasFill {
				for len(chunk) < size {
					chunk = append(chunk, fill)
				}
			}
			out = append(out, chunk)
		}
		return out
	case "slice":
		parts := 1
		if len(args) > 0 {
			parts = toInt(args[0], 1)
		}
		if parts <= 0 {
			parts = 1
		}
		arr := toSlice(v)
		if len(arr) == 0 {
			return []any{}
		}
		chunkSize := int(math.Ceil(float64(len(arr)) / float64(parts)))
		if chunkSize <= 0 {
			chunkSize = 1
		}
		out := []any{}
		for i := 0; i < len(arr); i += chunkSize {
			end := i + chunkSize
			if end > len(arr) {
				end = len(arr)
			}
			out = append(out, append([]any{}, arr[i:end]...))
		}
		return out
	case "sort":
		reverse := false
		caseSens := false
		attr := ""
		if len(args) > 0 {
			reverse = toBool(args[0], false)
		}
		if len(args) > 1 {
			caseSens = toBool(args[1], false)
		}
		if len(args) > 2 {
			attr = fmt.Sprint(args[2])
		}
		arr := append([]any{}, toSlice(v)...)
		sort.SliceStable(arr, func(i, j int) bool {
			a := arr[i]
			b := arr[j]
			if attr != "" {
				a = valueByPath(a, attr)
				b = valueByPath(b, attr)
			}
			cmp := compareAny(a, b, caseSens)
			if reverse {
				return cmp > 0
			}
			return cmp < 0
		})
		return arr
	case "dictsort":
		rv := reflect.ValueOf(v)
		if !rv.IsValid() || rv.Kind() != reflect.Map {
			return []any{}
		}
		by := "key"
		caseSens := false
		reverse := false
		if len(args) > 0 {
			if _, ok := args[0].(bool); ok {
				caseSens = toBool(args[0], false)
				if len(args) > 1 {
					by = strings.ToLower(fmt.Sprint(args[1]))
				}
				if len(args) > 2 {
					reverse = toBool(args[2], false)
				}
			} else {
				by = strings.ToLower(fmt.Sprint(args[0]))
				if len(args) > 1 {
					caseSens = toBool(args[1], false)
				}
				if len(args) > 2 {
					reverse = toBool(args[2], false)
				}
			}
		}
		if by != "key" && by != "value" {
			by = "key"
		}
		type pair struct {
			k any
			v any
		}
		pairs := make([]pair, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			pairs = append(pairs, pair{k: iter.Key().Interface(), v: iter.Value().Interface()})
		}
		sort.SliceStable(pairs, func(i, j int) bool {
			var lv any = pairs[i].k
			var rv any = pairs[j].k
			if by == "value" {
				lv = pairs[i].v
				rv = pairs[j].v
			}
			cmp := compareAny(lv, rv, caseSens)
			if reverse {
				return cmp > 0
			}
			return cmp < 0
		})
		out := make([]any, 0, len(pairs))
		for _, p := range pairs {
			out = append(out, []any{p.k, p.v})
		}
		return out
	case "groupby":
		attr := ""
		if len(args) > 0 {
			attr = fmt.Sprint(args[0])
		}
		var defaultVal any = missing
		hasDefault := false
		caseSens := false
		if len(args) > 1 {
			defaultVal = args[1]
			hasDefault = true
		}
		if len(args) > 2 {
			caseSens = toBool(args[2], false)
		}

		type grouped struct {
			grouper any
			norm    any
			list    []any
		}
		type groupItem struct {
			item any
			key  any
			norm any
		}

		arr := toSlice(v)
		items := make([]groupItem, 0, len(arr))
		for _, it := range arr {
			keyVal, ok := valueByPathEx(it, attr)
			if !ok {
				if hasDefault {
					keyVal = defaultVal
				} else {
					keyVal = missing
				}
			}
			normVal := keyVal
			if s, ok := keyVal.(string); ok && !caseSens {
				normVal = strings.ToLower(s)
			}
			items = append(items, groupItem{item: it, key: keyVal, norm: normVal})
		}

		sort.SliceStable(items, func(i, j int) bool {
			return compareAny(items[i].norm, items[j].norm, true) < 0
		})

		groups := map[string]*grouped{}
		for _, gi := range items {
			mapKey := fmt.Sprintf("%T|%v", gi.norm, gi.norm)
			g, ok := groups[mapKey]
			if !ok {
				g = &grouped{grouper: gi.key, norm: gi.norm, list: []any{}}
				groups[mapKey] = g
			}
			g.list = append(g.list, gi.item)
		}

		entries := make([]*grouped, 0, len(groups))
		for _, g := range groups {
			entries = append(entries, g)
		}
		sort.SliceStable(entries, func(i, j int) bool {
			return compareAny(entries[i].norm, entries[j].norm, true) < 0
		})

		out := make([]any, 0, len(entries))
		for _, g := range entries {
			out = append(out, map[string]any{
				"grouper": g.grouper,
				"list":    g.list,
			})
		}
		return out
	case "select":
		arr := toSlice(v)
		out := []any{}
		hasNeedle := len(args) > 0
		var needle any
		testName := ""
		testArgs := []any{}
		if hasNeedle {
			if name, ok := args[0].(string); ok && isKnownTestName(name) {
				testName = name
				testArgs = args[1:]
			} else {
				needle = args[0]
			}
		}
		for _, it := range arr {
			if testName != "" {
				if evalTestKeyword(it, testName, testArgs) {
					out = append(out, it)
				}
			} else if hasNeedle {
				if equalOp(it, needle) {
					out = append(out, it)
				}
			} else if truthy(it) {
				out = append(out, it)
			}
		}
		return out
	case "reject":
		arr := toSlice(v)
		out := []any{}
		hasNeedle := len(args) > 0
		var needle any
		testName := ""
		testArgs := []any{}
		if hasNeedle {
			if name, ok := args[0].(string); ok && isKnownTestName(name) {
				testName = name
				testArgs = args[1:]
			} else {
				needle = args[0]
			}
		}
		for _, it := range arr {
			if testName != "" {
				if !evalTestKeyword(it, testName, testArgs) {
					out = append(out, it)
				}
			} else if hasNeedle {
				if !equalOp(it, needle) {
					out = append(out, it)
				}
			} else if !truthy(it) {
				out = append(out, it)
			}
		}
		return out
	case "selectattr":
		attr := ""
		if len(args) > 0 {
			attr = fmt.Sprint(args[0])
		}
		arr := toSlice(v)
		out := []any{}
		hasNeedle := len(args) > 1
		var needle any
		testName := ""
		testArgs := []any{}
		if hasNeedle {
			if name, ok := args[1].(string); ok && isKnownTestName(name) {
				testName = name
				testArgs = args[2:]
			} else {
				needle = args[1]
			}
		}
		for _, it := range arr {
			av := valueByPath(it, attr)
			if testName != "" {
				if evalTestKeyword(av, testName, testArgs) {
					out = append(out, it)
				}
			} else if hasNeedle {
				if equalOp(av, needle) {
					out = append(out, it)
				}
			} else if truthy(av) {
				out = append(out, it)
			}
		}
		return out
	case "rejectattr":
		attr := ""
		if len(args) > 0 {
			attr = fmt.Sprint(args[0])
		}
		arr := toSlice(v)
		out := []any{}
		hasNeedle := len(args) > 1
		var needle any
		testName := ""
		testArgs := []any{}
		if hasNeedle {
			if name, ok := args[1].(string); ok && isKnownTestName(name) {
				testName = name
				testArgs = args[2:]
			} else {
				needle = args[1]
			}
		}
		for _, it := range arr {
			av := valueByPath(it, attr)
			if testName != "" {
				if !evalTestKeyword(av, testName, testArgs) {
					out = append(out, it)
				}
			} else if hasNeedle {
				if !equalOp(av, needle) {
					out = append(out, it)
				}
			} else if !truthy(av) {
				out = append(out, it)
			}
		}
		return out
	default:
		return v
	}
}

func invokeCallableValue(callable any, args []any, kwargs map[string]any, caller string) (any, error) {
	switch fn := callable.(type) {
	case TemplateFunc:
		return fn(args, kwargs, caller)
	case func(...any) any:
		return fn(args...), nil
	default:
		return "", fmt.Errorf("value is not callable")
	}
}

func invokeCallable(name string, argExprs []string, caller string, vars, ctx map[string]any) (any, error) {
	raw := resolveIdent(name, vars, ctx)
	posExprs, kwExprs := parseCallableArgs(argExprs)
	args := make([]any, 0, len(posExprs))
	kwargs := map[string]any{}
	for _, a := range posExprs {
		args = append(args, evalExpr(a, vars, ctx))
	}
	for k, expr := range kwExprs {
		kwargs[k] = evalExpr(expr, vars, ctx)
	}
	return invokeCallableValue(raw, args, kwargs, caller)
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return strings.TrimSpace(x) != ""
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	default:
		return true
	}
}

func numericOp(a any, b any, op string) any {
	fa := toFloat(a, 0)
	fb := toFloat(b, 0)
	switch op {
	case "+":
		if _, ok := a.(string); ok {
			return fmt.Sprint(a) + fmt.Sprint(b)
		}
		if _, ok := b.(string); ok {
			return fmt.Sprint(a) + fmt.Sprint(b)
		}
		return fa + fb
	case "-":
		return fa - fb
	case "*":
		return fa * fb
	case "/":
		if fb == 0 {
			return 0.0
		}
		return fa / fb
	case "%":
		ib := toInt(b, 0)
		if ib == 0 {
			return 0
		}
		return toInt(a, 0) % ib
	}
	return 0
}

func compareOp(a any, b any, op string) bool {
	switch op {
	case "==":
		return equalOp(a, b)
	case "!=":
		return !equalOp(a, b)
	case "<", "<=", ">", ">=":
		fa := toFloat(a, math.NaN())
		fb := toFloat(b, math.NaN())
		if !(math.IsNaN(fa) || math.IsNaN(fb)) {
			switch op {
			case "<":
				return fa < fb
			case "<=":
				return fa <= fb
			case ">":
				return fa > fb
			case ">=":
				return fa >= fb
			}
		}
		as := fmt.Sprint(a)
		bs := fmt.Sprint(b)
		switch op {
		case "<":
			return as < bs
		case "<=":
			return as <= bs
		case ">":
			return as > bs
		case ">=":
			return as >= bs
		}
	}
	return false
}

func equalOp(a any, b any) bool {
	if isMissing(a) && isMissing(b) {
		return true
	}
	if isNumber(a) && isNumber(b) {
		return toFloat(a, 0) == toFloat(b, 0)
	}
	return reflect.DeepEqual(a, b)
}

type exprTokenKind int

const (
	tokEOF exprTokenKind = iota
	tokNumber
	tokString
	tokIdent
	tokLParen
	tokRParen
	tokComma
	tokDot
	tokPipe
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokPercent
	tokEq
	tokNe
	tokLt
	tokLte
	tokGt
	tokGte
	tokAnd
	tokOr
	tokNot
	tokAssign
	tokIf
	tokElse
	tokIs
	tokIn
)

type exprToken struct {
	kind exprTokenKind
	lit  string
}

func lexExpr(src string) ([]exprToken, error) {
	tokens := make([]exprToken, 0, len(src)/2)
	i := 0
	for i < len(src) {
		ch := src[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			i++
			continue
		}
		if (ch >= '0' && ch <= '9') || ch == '.' {
			j := i
			dot := ch == '.'
			if ch == '.' {
				if i+1 >= len(src) || src[i+1] < '0' || src[i+1] > '9' {
					tokens = append(tokens, exprToken{kind: tokDot, lit: "."})
					i++
					continue
				}
			}
			for j < len(src) {
				c := src[j]
				if c >= '0' && c <= '9' {
					j++
					continue
				}
				if c == '.' && !dot {
					dot = true
					j++
					continue
				}
				break
			}
			tokens = append(tokens, exprToken{kind: tokNumber, lit: src[i:j]})
			i = j
			continue
		}
		if ch == '"' || ch == '\'' {
			q := ch
			j := i + 1
			for j < len(src) {
				if src[j] == '\\' {
					j += 2
					continue
				}
				if src[j] == q {
					break
				}
				j++
			}
			if j >= len(src) {
				return nil, fmt.Errorf("unterminated string")
			}
			tokens = append(tokens, exprToken{kind: tokString, lit: src[i+1 : j]})
			i = j + 1
			continue
		}
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_' {
			j := i + 1
			for j < len(src) {
				c := src[j]
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
					j++
					continue
				}
				break
			}
			lit := src[i:j]
			switch lit {
			case "and":
				tokens = append(tokens, exprToken{kind: tokAnd, lit: lit})
			case "or":
				tokens = append(tokens, exprToken{kind: tokOr, lit: lit})
			case "not":
				tokens = append(tokens, exprToken{kind: tokNot, lit: lit})
			case "is":
				tokens = append(tokens, exprToken{kind: tokIs, lit: lit})
			case "in":
				tokens = append(tokens, exprToken{kind: tokIn, lit: lit})
			case "if":
				tokens = append(tokens, exprToken{kind: tokIf, lit: lit})
			case "else":
				tokens = append(tokens, exprToken{kind: tokElse, lit: lit})
			default:
				tokens = append(tokens, exprToken{kind: tokIdent, lit: lit})
			}
			i = j
			continue
		}

		if i+1 < len(src) {
			two := src[i : i+2]
			switch two {
			case "==":
				tokens = append(tokens, exprToken{kind: tokEq, lit: two})
				i += 2
				continue
			case "!=":
				tokens = append(tokens, exprToken{kind: tokNe, lit: two})
				i += 2
				continue
			case "<=":
				tokens = append(tokens, exprToken{kind: tokLte, lit: two})
				i += 2
				continue
			case ">=":
				tokens = append(tokens, exprToken{kind: tokGte, lit: two})
				i += 2
				continue
			case "&&":
				tokens = append(tokens, exprToken{kind: tokAnd, lit: two})
				i += 2
				continue
			case "||":
				tokens = append(tokens, exprToken{kind: tokOr, lit: two})
				i += 2
				continue
			}
		}

		switch ch {
		case '(':
			tokens = append(tokens, exprToken{kind: tokLParen, lit: "("})
		case ')':
			tokens = append(tokens, exprToken{kind: tokRParen, lit: ")"})
		case ',':
			tokens = append(tokens, exprToken{kind: tokComma, lit: ","})
		case '.':
			tokens = append(tokens, exprToken{kind: tokDot, lit: "."})
		case '|':
			tokens = append(tokens, exprToken{kind: tokPipe, lit: "|"})
		case '+':
			tokens = append(tokens, exprToken{kind: tokPlus, lit: "+"})
		case '-':
			tokens = append(tokens, exprToken{kind: tokMinus, lit: "-"})
		case '*':
			tokens = append(tokens, exprToken{kind: tokStar, lit: "*"})
		case '/':
			tokens = append(tokens, exprToken{kind: tokSlash, lit: "/"})
		case '%':
			tokens = append(tokens, exprToken{kind: tokPercent, lit: "%"})
		case '<':
			tokens = append(tokens, exprToken{kind: tokLt, lit: "<"})
		case '>':
			tokens = append(tokens, exprToken{kind: tokGt, lit: ">"})
		case '!':
			tokens = append(tokens, exprToken{kind: tokNot, lit: "!"})
		case '=':
			tokens = append(tokens, exprToken{kind: tokAssign, lit: "="})
		default:
			return nil, fmt.Errorf("unexpected token: %c", ch)
		}
		i++
	}
	tokens = append(tokens, exprToken{kind: tokEOF})
	return tokens, nil
}

type exprParser struct {
	toks []exprToken
	pos  int
	vars map[string]any
	ctx  map[string]any
}

func (p *exprParser) cur() exprToken {
	if p.pos >= len(p.toks) {
		return exprToken{kind: tokEOF}
	}
	return p.toks[p.pos]
}

func (p *exprParser) next() exprToken {
	if p.pos+1 >= len(p.toks) {
		return exprToken{kind: tokEOF}
	}
	return p.toks[p.pos+1]
}

func (p *exprParser) advance() {
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
}

func (p *exprParser) match(k exprTokenKind) bool {
	if p.cur().kind == k {
		p.advance()
		return true
	}
	return false
}

func (p *exprParser) expect(k exprTokenKind) error {
	if p.cur().kind != k {
		return fmt.Errorf("expected %v, got %v", k, p.cur().kind)
	}
	p.advance()
	return nil
}

func (p *exprParser) parseExpression() (any, error) {
	return p.parseConditional()
}

func (p *exprParser) parseConditional() (any, error) {
	left, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.match(tokIf) {
		cond, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(tokElse); err != nil {
			return nil, err
		}
		right, err := p.parseConditional()
		if err != nil {
			return nil, err
		}
		if truthy(cond) {
			return left, nil
		}
		return right, nil
	}
	return left, nil
}

func (p *exprParser) parseOr() (any, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.cur().kind == tokOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = truthy(left) || truthy(right)
	}
	return left, nil
}

func (p *exprParser) parseAnd() (any, error) {
	left, err := p.parseCompare()
	if err != nil {
		return nil, err
	}
	for p.cur().kind == tokAnd {
		p.advance()
		right, err := p.parseCompare()
		if err != nil {
			return nil, err
		}
		left = truthy(left) && truthy(right)
	}
	return left, nil
}

func (p *exprParser) parseCompare() (any, error) {
	left, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	prev := left
	hasCmp := false
	result := true

	for {
		op := p.cur().kind
		if op != tokEq && op != tokNe && op != tokLt && op != tokLte && op != tokGt && op != tokGte && op != tokIs && op != tokIn && !(op == tokNot && p.next().kind == tokIn) {
			break
		}
		hasCmp = true
		opLit := p.cur().lit
		isNot := false
		if op == tokNot && p.next().kind == tokIn {
			p.advance()
			p.advance()
			opLit = "not in"
		} else if op == tokIs {
			p.advance()
			if p.cur().kind == tokNot {
				isNot = true
				p.advance()
			}
			if p.cur().kind == tokIdent {
				testKw := p.cur().lit
				if isKnownTestName(testKw) {
					p.advance()
					testArgs := []any{}
					if p.cur().kind == tokLParen {
						p.advance()
						args, kwargs, err := p.parseCallArgs()
						if err != nil {
							return nil, err
						}
						if len(kwargs) > 0 {
							return nil, fmt.Errorf("named args in tests not supported")
						}
						if err := p.expect(tokRParen); err != nil {
							return nil, err
						}
						testArgs = args
					}
					ok := evalTestKeyword(prev, testKw, testArgs)
					if isNot {
						ok = !ok
					}
					result = result && ok
					// preserve chain semantics: a is defined is true compares bool->right next
					prev = ok
					continue
				}
			}
			// fallback: "a is b"
			right, err := p.parseAdd()
			if err != nil {
				return nil, err
			}
			ok := compareOp(prev, right, "==")
			if isNot {
				ok = !ok
			}
			result = result && ok
			prev = right
			continue
		} else {
			p.advance()
		}
		right, err := p.parseAdd()
		if err != nil {
			return nil, err
		}
		var ok bool
		switch opLit {
		case "in":
			ok = containsOp(right, prev)
		case "not in":
			ok = !containsOp(right, prev)
		default:
			ok = compareOp(prev, right, opLit)
		}
		result = result && ok
		prev = right
	}
	if hasCmp {
		return result, nil
	}
	return left, nil
}

func (p *exprParser) parseAdd() (any, error) {
	left, err := p.parseMul()
	if err != nil {
		return nil, err
	}
	for p.cur().kind == tokPlus || p.cur().kind == tokMinus {
		op := p.cur().lit
		p.advance()
		right, err := p.parseMul()
		if err != nil {
			return nil, err
		}
		left = numericOp(left, right, op)
	}
	return left, nil
}

func (p *exprParser) parseMul() (any, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.cur().kind == tokStar || p.cur().kind == tokSlash || p.cur().kind == tokPercent {
		op := p.cur().lit
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = numericOp(left, right, op)
	}
	return left, nil
}

func (p *exprParser) parseUnary() (any, error) {
	if p.cur().kind == tokNot {
		p.advance()
		v, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return !truthy(v), nil
	}
	if p.cur().kind == tokMinus {
		p.advance()
		v, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return -toFloat(v, 0), nil
	}
	if p.cur().kind == tokPlus {
		p.advance()
		return p.parseUnary()
	}
	return p.parsePostfix()
}

func (p *exprParser) parseCallArgs() ([]any, map[string]any, error) {
	args := []any{}
	kwargs := map[string]any{}
	if p.cur().kind == tokRParen {
		return args, kwargs, nil
	}
	for {
		if p.cur().kind == tokIdent && p.next().kind == tokAssign {
			name := p.cur().lit
			p.advance()
			p.advance()
			v, err := p.parseExpression()
			if err != nil {
				return nil, nil, err
			}
			kwargs[name] = v
		} else {
			v, err := p.parseExpression()
			if err != nil {
				return nil, nil, err
			}
			args = append(args, v)
		}
		if p.cur().kind != tokComma {
			break
		}
		p.advance()
	}
	return args, kwargs, nil
}

func (p *exprParser) parsePostfix() (any, error) {
	val, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch p.cur().kind {
		case tokDot:
			p.advance()
			if p.cur().kind != tokIdent {
				return nil, fmt.Errorf("expected identifier after dot")
			}
			key := p.cur().lit
			p.advance()
			switch m := val.(type) {
			case map[string]any:
				val = m[key]
			default:
				val = ""
			}
		case tokLParen:
			p.advance()
			args, kwargs, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			if err := p.expect(tokRParen); err != nil {
				return nil, err
			}
			called, err := invokeCallableValue(val, args, kwargs, "")
			if err != nil {
				return nil, err
			}
			val = called
		case tokPipe:
			p.advance()
			if p.cur().kind != tokIdent {
				return nil, fmt.Errorf("expected filter name")
			}
			fname := p.cur().lit
			p.advance()
			fargs := []any{}
			if p.cur().kind == tokLParen {
				p.advance()
				args, kwargs, err := p.parseCallArgs()
				if err != nil {
					return nil, err
				}
				if len(kwargs) > 0 {
					return nil, fmt.Errorf("named args in filters not supported")
				}
				fargs = args
				if err := p.expect(tokRParen); err != nil {
					return nil, err
				}
			}
			val = applyFilter(fname, val, fargs)
		default:
			return val, nil
		}
	}
}

func (p *exprParser) parsePrimary() (any, error) {
	t := p.cur()
	switch t.kind {
	case tokNumber:
		p.advance()
		if strings.Contains(t.lit, ".") {
			f, _ := strconv.ParseFloat(t.lit, 64)
			return f, nil
		}
		i, _ := strconv.Atoi(t.lit)
		return i, nil
	case tokString:
		p.advance()
		return t.lit, nil
	case tokIdent:
		p.advance()
		switch t.lit {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null", "nil":
			return nil, nil
		default:
			return resolveIdent(t.lit, p.vars, p.ctx), nil
		}
	case tokLParen:
		p.advance()
		v, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expect(tokRParen); err != nil {
			return nil, err
		}
		return v, nil
	default:
		return "", fmt.Errorf("unexpected token in expression")
	}
}

func evalExpr(expr string, vars, ctx map[string]any) any {
	toks, err := lexExpr(expr)
	if err != nil {
		if lit, ok := parseLiteral(strings.TrimSpace(expr)); ok {
			return lit
		}
		return resolveIdent(strings.TrimSpace(expr), vars, ctx)
	}
	p := &exprParser{toks: toks, vars: vars, ctx: ctx}
	v, err := p.parseExpression()
	if err != nil {
		if lit, ok := parseLiteral(strings.TrimSpace(expr)); ok {
			return lit
		}
		return resolveIdent(strings.TrimSpace(expr), vars, ctx)
	}
	return v
}

func evalCond(cond string, vars, ctx map[string]any) bool {
	return truthy(evalExpr(cond, vars, ctx))
}
