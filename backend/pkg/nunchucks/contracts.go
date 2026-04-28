package nunchucks

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var contractCommentRe = regexp.MustCompile(`\{#([\s\S]*?)#\}`)

type ContractType struct {
	Name   string
	Fields map[string]ContractProp
}

type ContractProp struct {
	Name        string
	Type        ContractType
	Optional    bool
	HasDefault  bool
	DefaultExpr string
}

type TemplateContract struct {
	Props  map[string]ContractProp
	Params map[string]ContractType
}

func ParseTemplateContract(src string) (TemplateContract, error) {
	contract := TemplateContract{
		Props:  map[string]ContractProp{},
		Params: map[string]ContractType{},
	}

	matches := contractCommentRe.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		body := strings.TrimSpace(match[1])
		if body == "" {
			continue
		}

		lines := strings.Split(body, "\n")
		header := strings.TrimSpace(lines[0])
		rest := ""
		if len(lines) > 1 {
			rest = strings.Join(lines[1:], "\n")
		}

		switch {
		case header == "@props":
			props, err := parseContractProps(rest, contract.Params)
			if err != nil {
				return TemplateContract{}, err
			}
			for name, prop := range props {
				contract.Props[name] = prop
			}
		case strings.HasPrefix(header, "@params "):
			name := strings.TrimSpace(strings.TrimPrefix(header, "@params "))
			if name == "" {
				return TemplateContract{}, fmt.Errorf("invalid @params declaration")
			}
			fields, err := parseContractProps(rest, contract.Params)
			if err != nil {
				return TemplateContract{}, err
			}
			contract.Params[name] = ContractType{
				Name:   name,
				Fields: fields,
			}
		}
	}

	return contract, nil
}

func ValidateTemplateContract(src string, scope map[string]any) error {
	contract, err := ParseTemplateContract(src)
	if err != nil {
		return err
	}
	if len(contract.Props) == 0 {
		return nil
	}

	errs := make([]string, 0)
	for name, prop := range contract.Props {
		value, ok := scope[name]
		if (!ok || value == nil) && !prop.Optional && !prop.HasDefault {
			errs = append(errs, fmt.Sprintf("missing required prop %q", name))
			continue
		}
		if !ok || value == nil {
			continue
		}
		if err := validateContractValue(name, value, prop.Type); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("template contract validation failed: %s", strings.Join(errs, "; "))
}

func ApplyTemplateContractDefaults(src string, scope map[string]any) error {
	contract, err := ParseTemplateContract(src)
	if err != nil {
		return err
	}
	if len(contract.Props) == 0 {
		return nil
	}

	for name, prop := range contract.Props {
		value, ok := scope[name]
		if (!ok || value == nil) && prop.HasDefault {
			resolved, err := evaluateContractDefault(prop.DefaultExpr, scope)
			if err != nil {
				return fmt.Errorf("invalid default for %q: %w", name, err)
			}
			scope[name] = resolved
			value = resolved
			ok = true
		}
		if !ok || value == nil {
			continue
		}
		next, err := applyContractValueDefaults(value, prop.Type, scope)
		if err != nil {
			return fmt.Errorf("invalid default for %q: %w", name, err)
		}
		if next != nil {
			scope[name] = next
		}
	}

	return nil
}

func parseContractProps(body string, params map[string]ContractType) (map[string]ContractProp, error) {
	props := map[string]ContractProp{}
	for _, raw := range splitContractEntries(body) {
		if raw == "" {
			continue
		}
		prop, err := parseContractProp(raw, params)
		if err != nil {
			return nil, err
		}
		props[prop.Name] = prop
	}
	return props, nil
}

func splitContractEntries(body string) []string {
	lines := strings.Split(body, "\n")
	out := make([]string, 0)
	var current strings.Builder
	depth := 0

	flush := func() {
		entry := strings.TrimSpace(current.String())
		if entry != "" {
			out = append(out, entry)
		}
		current.Reset()
		depth = 0
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(trimmed)
		depth += strings.Count(trimmed, "{")
		depth -= strings.Count(trimmed, "}")
		if depth <= 0 {
			flush()
		}
	}
	flush()
	return out
}

func parseContractProp(raw string, params map[string]ContractType) (ContractProp, error) {
	left, right, ok := splitTopLevelColon(raw)
	if !ok {
		return ContractProp{}, fmt.Errorf("invalid contract field: %s", raw)
	}

	name := strings.TrimSpace(left)
	optional := false
	if strings.HasSuffix(name, "?") {
		optional = true
		name = strings.TrimSuffix(name, "?")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ContractProp{}, fmt.Errorf("invalid contract field: %s", raw)
	}

	typeExpr := strings.TrimSpace(right)
	defaultExpr := ""
	if _, _, ok := splitTopLevelAssign(typeExpr); ok {
		eq := strings.Index(typeExpr, "=")
		defaultExpr = strings.TrimSpace(typeExpr[eq+1:])
		typeExpr = strings.TrimSpace(typeExpr[:eq])
	}

	typ, err := parseContractType(typeExpr, params)
	if err != nil {
		return ContractProp{}, fmt.Errorf("invalid type for %s: %w", name, err)
	}

	return ContractProp{
		Name:        name,
		Type:        typ,
		Optional:    optional,
		HasDefault:  defaultExpr != "",
		DefaultExpr: defaultExpr,
	}, nil
}

func parseContractType(raw string, params map[string]ContractType) (ContractType, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return ContractType{}, fmt.Errorf("missing type")
	}
	if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
		body := strings.TrimSpace(t[1 : len(t)-1])
		fields, err := parseInlineObjectFields(body, params)
		if err != nil {
			return ContractType{}, err
		}
		return ContractType{Name: "object", Fields: fields}, nil
	}

	switch strings.ToLower(t) {
	case "string", "number", "int", "float", "bool", "object", "list", "array":
		name := strings.ToLower(t)
		if name == "array" {
			name = "list"
		}
		return ContractType{Name: name}, nil
	}

	if named, ok := params[t]; ok {
		return named, nil
	}

	return ContractType{}, fmt.Errorf("unknown type %q", t)
}

func parseInlineObjectFields(body string, params map[string]ContractType) (map[string]ContractProp, error) {
	parts := splitTopLevelObjectFields(body)
	fields := map[string]ContractProp{}
	for _, part := range parts {
		prop, err := parseContractProp(part, params)
		if err != nil {
			return nil, err
		}
		fields[prop.Name] = prop
	}
	return fields, nil
}

func splitTopLevelObjectFields(src string) []string {
	out := make([]string, 0)
	var cur strings.Builder
	depth := 0
	quote := byte(0)
	esc := false

	flush := func() {
		part := strings.TrimSpace(cur.String())
		if part != "" {
			out = append(out, part)
		}
		cur.Reset()
	}

	for i := 0; i < len(src); i++ {
		ch := src[i]
		if esc {
			cur.WriteByte(ch)
			esc = false
			continue
		}
		if quote != 0 {
			cur.WriteByte(ch)
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
			cur.WriteByte(ch)
			continue
		}
		switch ch {
		case '{', '[', '(':
			depth++
			cur.WriteByte(ch)
		case '}', ']', ')':
			if depth > 0 {
				depth--
			}
			cur.WriteByte(ch)
		case ',', '\n':
			if depth == 0 {
				flush()
				continue
			}
			cur.WriteByte(ch)
		default:
			cur.WriteByte(ch)
		}
	}
	flush()
	return out
}

func splitTopLevelColon(src string) (string, string, bool) {
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
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				left := strings.TrimSpace(s[:i])
				right := strings.TrimSpace(s[i+1:])
				if left != "" && right != "" {
					return left, right, true
				}
				return "", "", false
			}
		}
	}
	return "", "", false
}

func validateContractValue(path string, value any, typ ContractType) error {
	if len(typ.Fields) > 0 {
		obj, ok := toObjectMap(value)
		if !ok {
			return fmt.Errorf("prop %q should be object, got %T", path, value)
		}
		for name, field := range typ.Fields {
			fieldValue, exists := obj[name]
			if (!exists || fieldValue == nil) && !field.Optional && !field.HasDefault {
				return fmt.Errorf("missing required prop %q", path+"."+name)
			}
			if !exists || fieldValue == nil {
				continue
			}
			if err := validateContractValue(path+"."+name, fieldValue, field.Type); err != nil {
				return err
			}
		}
		return nil
	}

	switch typ.Name {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("prop %q should be string, got %T", path, value)
		}
	case "number":
		if !isNumericType(value) {
			return fmt.Errorf("prop %q should be number, got %T", path, value)
		}
	case "int":
		if !isIntegerType(value) {
			return fmt.Errorf("prop %q should be int, got %T", path, value)
		}
	case "float":
		if !isFloatType(value) {
			return fmt.Errorf("prop %q should be float, got %T", path, value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("prop %q should be bool, got %T", path, value)
		}
	case "list":
		if !isListType(value) {
			return fmt.Errorf("prop %q should be list, got %T", path, value)
		}
	case "object":
		if !isObjectType(value) {
			return fmt.Errorf("prop %q should be object, got %T", path, value)
		}
	default:
		return fmt.Errorf("unsupported contract type %q", typ.Name)
	}
	return nil
}

func applyContractValueDefaults(value any, typ ContractType, scope map[string]any) (any, error) {
	if len(typ.Fields) == 0 {
		return value, nil
	}

	obj, ok := toObjectMap(value)
	if !ok {
		return nil, nil
	}

	for name, field := range typ.Fields {
		fieldValue, exists := obj[name]
		if (!exists || fieldValue == nil) && field.HasDefault {
			resolved, err := evaluateContractDefault(field.DefaultExpr, obj)
			if err != nil {
				return nil, err
			}
			obj[name] = resolved
			fieldValue = resolved
			exists = true
		}
		if !exists || fieldValue == nil {
			continue
		}
		next, err := applyContractValueDefaults(fieldValue, field.Type, obj)
		if err != nil {
			return nil, err
		}
		if next != nil {
			obj[name] = next
		}
	}

	return obj, nil
}

func evaluateContractDefault(expr string, scope map[string]any) (any, error) {
	raw := strings.TrimSpace(expr)
	if raw == "" {
		return nil, fmt.Errorf("missing default expression")
	}

	if strings.HasPrefix(raw, "default(") && strings.HasSuffix(raw, ")") {
		raw = strings.TrimSpace(raw[len("default(") : len(raw)-1])
	}

	value := evalExpr(raw, scope, scope)
	if isMissing(value) {
		return nil, fmt.Errorf("unresolved expression %q", expr)
	}
	return value, nil
}

func isNumericType(v any) bool {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return false
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isIntegerType(v any) bool {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return false
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloatType(v any) bool {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return false
	}
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isListType(v any) bool {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return false
	}
	return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
}

func isObjectType(v any) bool {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return false
	}
	return rv.Kind() == reflect.Map || rv.Kind() == reflect.Struct
}

func toObjectMap(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, false
	}
	if rv.Kind() == reflect.Map {
		out := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key()
			if key.Kind() != reflect.String {
				return nil, false
			}
			out[key.String()] = iter.Value().Interface()
		}
		return out, true
	}
	if rv.Kind() == reflect.Struct {
		out := map[string]any{}
		rt := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			if !rv.Field(i).CanInterface() {
				continue
			}
			out[rt.Field(i).Name] = rv.Field(i).Interface()
		}
		return out, true
	}
	return nil, false
}
