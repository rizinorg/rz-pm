package envparse

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Unmarshal(input string, out any) error {
	data := parseKeyValueString(input)

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("output must be a pointer to a struct")
	}

	v = v.Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("env")
		if tag == "" {
			continue
		}

		valueStr, ok := data[tag]
		if !ok {
			continue
		}

		fieldVal := v.Field(i)
		if !fieldVal.CanSet() {
			continue
		}

		switch fieldVal.Kind() {
		case reflect.String:
			fieldVal.SetString(valueStr)
		case reflect.Int, reflect.Int64:
			intVal, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing int for field %s: %w", field.Name, err)
			}
			fieldVal.SetInt(intVal)
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(valueStr)
			if err != nil {
				return fmt.Errorf("error parsing bool for field %s: %w", field.Name, err)
			}
			fieldVal.SetBool(boolVal)
		default:
			return fmt.Errorf("unsupported type %s for field %s", fieldVal.Kind(), field.Name)
		}
	}

	return nil
}

// Parses a KEY=value formatted string
func parseKeyValueString(input string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		result[key] = val
	}
	return result
}
