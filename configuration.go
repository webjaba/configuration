package configuration

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const defaultEnvPath = ".env"

var durationType = reflect.TypeFor[time.Duration]()

type Opts interface {
	apply(*options)
}

type WithPathOption struct {
	Path string
}

func (o WithPathOption) apply(opts *options) {
	if o.Path != "" {
		opts.path = o.Path
	}
}

type options struct {
	path string
}

func New(cfg any, opts ...Opts) error {
	parsedOptions := options{path: defaultEnvPath}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&parsedOptions)
		}
	}

	value := reflect.ValueOf(cfg)
	if !value.IsValid() {
		return errors.New("configuration: expected pointer to struct")
	}
	if value.Kind() != reflect.Pointer {
		return errors.New("configuration: expected pointer to struct")
	}
	if value.IsNil() {
		return errors.New("configuration: nil config pointer")
	}

	structValue := value.Elem()
	if structValue.Kind() != reflect.Struct {
		return errors.New("configuration: expected pointer to struct")
	}

	values, err := parseEnvFile(parsedOptions.path)
	if err != nil {
		return err
	}

	return fillStruct(structValue, values, structValue.Type().Name())
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("configuration: .env file not found: %s", path)
		}
		return nil, fmt.Errorf("configuration: cannot open .env file %q: %w", path, err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("configuration: invalid .env line %d: missing '='", lineNumber)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("configuration: invalid .env line %d: empty key", lineNumber)
		}

		value = strings.TrimSpace(value)
		value = trimQuotes(value)
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("configuration: cannot read .env file %q: %w", path, err)
	}

	return values, nil
}

func trimQuotes(value string) string {
	if len(value) < 2 {
		return value
	}

	if value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}
	if value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}

	return value
}

func fillStruct(value reflect.Value, envValues map[string]string, path string) error {
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := valueType.Field(i)
		fieldPath := path + "." + fieldType.Name

		if !fieldType.IsExported() {
			continue
		}

		envName := fieldType.Tag.Get("env")
		if envName == "-" {
			continue
		}

		if isNestedStruct(field) {
			if err := fillStruct(field, envValues, fieldPath); err != nil {
				return err
			}
			continue
		}

		if envName == "" {
			return fmt.Errorf("configuration: missing env tag for field %s", fieldPath)
		}

		raw, ok := envValues[envName]
		if !ok {
			return fmt.Errorf("configuration: missing env variable %q for field %s", envName, fieldPath)
		}
		if raw == "" {
			return fmt.Errorf("configuration: empty value for env variable %q", envName)
		}

		if err := setField(field, envName, raw); err != nil {
			return fmt.Errorf("configuration: cannot parse %q for field %s: %w", envName, fieldPath, err)
		}
	}

	return nil
}

func isNestedStruct(value reflect.Value) bool {
	return value.Kind() == reflect.Struct && value.Type() != durationType
}

func setField(field reflect.Value, envName string, raw string) error {
	if !field.CanSet() {
		return errors.New("field cannot be set")
	}

	if field.Type() == durationType {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}
		field.SetInt(int64(parsed))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(parsed)
	default:
		return fmt.Errorf("unsupported field type %s for env variable %q", field.Type(), envName)
	}

	return nil
}
