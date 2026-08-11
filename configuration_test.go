package configuration

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any))
		wantErr string
	}{
		{
			name: "parses supported types and nested structs",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				envPath := writeEnv(t, `
STRING=value
BOOL=true
INT=-1
INT8=-8
INT16=-16
INT32=-32
INT64=-64
UINT=1
UINT8=8
UINT16=16
UINT32=32
UINT64=64
FLOAT32=1.25
FLOAT64=2.5
DURATION=5s
NESTED=value
`)

				type Nested struct {
					Value string `env:"NESTED"`
				}
				type Config struct {
					String   string        `env:"STRING"`
					Bool     bool          `env:"BOOL"`
					Int      int           `env:"INT"`
					Int8     int8          `env:"INT8"`
					Int16    int16         `env:"INT16"`
					Int32    int32         `env:"INT32"`
					Int64    int64         `env:"INT64"`
					Uint     uint          `env:"UINT"`
					Uint8    uint8         `env:"UINT8"`
					Uint16   uint16        `env:"UINT16"`
					Uint32   uint32        `env:"UINT32"`
					Uint64   uint64        `env:"UINT64"`
					Float32  float32       `env:"FLOAT32"`
					Float64  float64       `env:"FLOAT64"`
					Duration time.Duration `env:"DURATION"`
					Nested   Nested
					Ignored  string `env:"-"`
					hidden   string
				}

				cfg := &Config{}
				assert := func(t *testing.T, cfg any) {
					t.Helper()

					got := cfg.(*Config)
					if got.String != "value" || !got.Bool || got.Int != -1 || got.Int8 != -8 || got.Int16 != -16 || got.Int32 != -32 || got.Int64 != -64 {
						t.Fatalf("signed/string/bool values were not parsed correctly: %+v", got)
					}
					if got.Uint != 1 || got.Uint8 != 8 || got.Uint16 != 16 || got.Uint32 != 32 || got.Uint64 != 64 {
						t.Fatalf("unsigned values were not parsed correctly: %+v", got)
					}
					if got.Float32 != 1.25 || got.Float64 != 2.5 {
						t.Fatalf("float values were not parsed correctly: %+v", got)
					}
					if got.Duration != 5*time.Second {
						t.Fatalf("Duration = %s, want 5s", got.Duration)
					}
					if got.Nested.Value != "value" {
						t.Fatalf("Nested.Value = %q, want value", got.Nested.Value)
					}
					if got.Ignored != "" || got.hidden != "" {
						t.Fatalf("ignored or hidden fields were modified: %+v", got)
					}
				}

				return cfg, []Opts{WithPathOption{Path: envPath}}, assert
			},
		},
		{
			name: "uses default env path with empty path option and ignores nil option",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				tmpDir := t.TempDir()
				writeEnvAt(t, filepath.Join(tmpDir, defaultEnvPath), "VALUE=ok\n")
				t.Chdir(tmpDir)

				var nilOpt Opts
				cfg := &struct {
					Value string `env:"VALUE"`
				}{}

				return cfg, []Opts{nilOpt, WithPathOption{Path: ""}}, func(t *testing.T, cfg any) {
					t.Helper()
					got := cfg.(*struct {
						Value string `env:"VALUE"`
					})
					if got.Value != "ok" {
						t.Fatalf("Value = %q, want ok", got.Value)
					}
				}
			},
		},
		{
			name: "nil config",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				return nil, nil, nil
			},
			wantErr: "expected pointer to struct",
		},
		{
			name: "not pointer",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				return struct{}{}, nil, nil
			},
			wantErr: "expected pointer to struct",
		},
		{
			name: "nil pointer",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				return (*struct{})(nil), nil, nil
			},
			wantErr: "nil config pointer",
		},
		{
			name: "not struct pointer",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				return ptr("test"), nil, nil
			},
			wantErr: "expected pointer to struct",
		},
		{
			name: "missing file",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				return &struct{}{}, []Opts{WithPathOption{Path: filepath.Join(t.TempDir(), ".env")}}, nil
			},
			wantErr: ".env file not found",
		},
		{
			name: "missing env tag",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				return &struct{ Value string }{}, []Opts{WithPathOption{Path: writeEnv(t, "VALUE=test\n")}}, nil
			},
			wantErr: "missing env tag",
		},
		{
			name: "missing env variable",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				cfg := &struct {
					Value string `env:"VALUE"`
				}{}
				return cfg, []Opts{WithPathOption{Path: writeEnv(t, "OTHER=test\n")}}, nil
			},
			wantErr: "missing env variable \"VALUE\"",
		},
		{
			name: "empty value",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				cfg := &struct {
					Value string `env:"VALUE"`
				}{}
				return cfg, []Opts{WithPathOption{Path: writeEnv(t, "VALUE=\n")}}, nil
			},
			wantErr: "empty value",
		},
		{
			name: "unsupported type",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				cfg := &struct {
					Values []string `env:"VALUES"`
				}{}
				return cfg, []Opts{WithPathOption{Path: writeEnv(t, "VALUES=a,b\n")}}, nil
			},
			wantErr: "unsupported field type []string",
		},
		{
			name: "invalid bool",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				cfg := &struct {
					Value bool `env:"VALUE"`
				}{}
				return cfg, []Opts{WithPathOption{Path: writeEnv(t, "VALUE=invalid\n")}}, nil
			},
			wantErr: "cannot parse \"VALUE\"",
		},
		{
			name: "invalid int",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				cfg := &struct {
					Value int `env:"VALUE"`
				}{}
				return cfg, []Opts{WithPathOption{Path: writeEnv(t, "VALUE=invalid\n")}}, nil
			},
			wantErr: "cannot parse \"VALUE\"",
		},
		{
			name: "invalid uint",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				cfg := &struct {
					Value uint `env:"VALUE"`
				}{}
				return cfg, []Opts{WithPathOption{Path: writeEnv(t, "VALUE=-1\n")}}, nil
			},
			wantErr: "cannot parse \"VALUE\"",
		},
		{
			name: "invalid float",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				cfg := &struct {
					Value float64 `env:"VALUE"`
				}{}
				return cfg, []Opts{WithPathOption{Path: writeEnv(t, "VALUE=invalid\n")}}, nil
			},
			wantErr: "cannot parse \"VALUE\"",
		},
		{
			name: "invalid duration",
			arrange: func(t *testing.T) (any, []Opts, func(t *testing.T, cfg any)) {
				cfg := &struct {
					Value time.Duration `env:"VALUE"`
				}{}
				return cfg, []Opts{WithPathOption{Path: writeEnv(t, "VALUE=invalid\n")}}, nil
			},
			wantErr: "cannot parse \"VALUE\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, opts, assert := tt.arrange(t)
			err := New(cfg, opts...)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("New() error = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("New() error = %q, want to contain %q", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if assert != nil {
				assert(t, cfg)
			}
		})
	}
}

func TestParseEnvFile(t *testing.T) {
	tests := []struct {
		name    string
		path    func(t *testing.T) string
		want    map[string]string
		wantErr string
	}{
		{
			name: "parses supported env syntax",
			path: func(t *testing.T) string {
				return writeEnv(t, `
# comment

KEY=value
 SPACED = trimmed 
DOUBLE="quoted value"
SINGLE='quoted value'
URL=http://localhost?a=b
`)
			},
			want: map[string]string{
				"KEY":    "value",
				"SPACED": "trimmed",
				"DOUBLE": "quoted value",
				"SINGLE": "quoted value",
				"URL":    "http://localhost?a=b",
			},
		},
		{
			name: "missing file",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), ".env")
			},
			wantErr: ".env file not found",
		},
		{
			name: "missing separator",
			path: func(t *testing.T) string {
				return writeEnv(t, "INVALID\n")
			},
			wantErr: "missing '='",
		},
		{
			name: "empty key",
			path: func(t *testing.T) string {
				return writeEnv(t, "=value\n")
			},
			wantErr: "empty key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEnvFile(tt.path(t))

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("parseEnvFile() error = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseEnvFile() error = %q, want to contain %q", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseEnvFile() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseEnvFile() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: ""},
		{name: "one character", value: "a", want: "a"},
		{name: "without quotes", value: "value", want: "value"},
		{name: "double quotes", value: "\"value\"", want: "value"},
		{name: "single quotes", value: "'value'", want: "value"},
		{name: "mismatched quotes", value: "\"value'", want: "\"value'"},
		{name: "empty double quoted", value: "\"\"", want: ""},
		{name: "empty single quoted", value: "''", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimQuotes(tt.value); got != tt.want {
				t.Fatalf("trimQuotes(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsNestedStruct(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "struct", value: struct{}{}, want: true},
		{name: "duration", value: time.Duration(0), want: false},
		{name: "string", value: "value", want: false},
		{name: "int", value: 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNestedStruct(reflect.ValueOf(tt.value)); got != tt.want {
				t.Fatalf("isNestedStruct(%T) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}

func TestSetField(t *testing.T) {
	type Fields struct {
		String   string
		Bool     bool
		Int      int
		Int8     int8
		Int16    int16
		Int32    int32
		Int64    int64
		Uint     uint
		Uint8    uint8
		Uint16   uint16
		Uint32   uint32
		Uint64   uint64
		Float32  float32
		Float64  float64
		Duration time.Duration
		Slice    []string
	}

	tests := []struct {
		name      string
		field     func(fields *Fields) reflect.Value
		raw       string
		assert    func(t *testing.T, fields Fields)
		wantError string
	}{
		{name: "string", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("String") }, raw: "value", assert: func(t *testing.T, f Fields) { assertEqual(t, f.String, "value") }},
		{name: "bool", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Bool") }, raw: "true", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Bool, true) }},
		{name: "int", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Int") }, raw: "-1", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Int, -1) }},
		{name: "int8", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Int8") }, raw: "-8", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Int8, int8(-8)) }},
		{name: "int16", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Int16") }, raw: "-16", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Int16, int16(-16)) }},
		{name: "int32", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Int32") }, raw: "-32", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Int32, int32(-32)) }},
		{name: "int64", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Int64") }, raw: "-64", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Int64, int64(-64)) }},
		{name: "uint", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Uint") }, raw: "1", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Uint, uint(1)) }},
		{name: "uint8", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Uint8") }, raw: "8", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Uint8, uint8(8)) }},
		{name: "uint16", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Uint16") }, raw: "16", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Uint16, uint16(16)) }},
		{name: "uint32", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Uint32") }, raw: "32", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Uint32, uint32(32)) }},
		{name: "uint64", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Uint64") }, raw: "64", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Uint64, uint64(64)) }},
		{name: "float32", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Float32") }, raw: "1.25", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Float32, float32(1.25)) }},
		{name: "float64", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Float64") }, raw: "2.5", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Float64, 2.5) }},
		{name: "duration", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Duration") }, raw: "5s", assert: func(t *testing.T, f Fields) { assertEqual(t, f.Duration, 5*time.Second) }},
		{name: "unsettable field", field: func(f *Fields) reflect.Value { return reflect.ValueOf(*f).FieldByName("String") }, raw: "value", wantError: "field cannot be set"},
		{name: "unsupported type", field: func(f *Fields) reflect.Value { return reflect.ValueOf(f).Elem().FieldByName("Slice") }, raw: "a,b", wantError: "unsupported field type []string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fields Fields
			err := setField(tt.field(&fields), "VALUE", tt.raw)

			if tt.wantError != "" {
				if err == nil {
					t.Fatal("setField() error = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("setField() error = %q, want to contain %q", err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Fatalf("setField() error = %v", err)
			}
			if tt.assert != nil {
				tt.assert(t, fields)
			}
		})
	}
}

func writeEnv(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	writeEnvAt(t, path, strings.TrimLeft(content, "\n"))

	return path
}

func writeEnvAt(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
}

func assertEqual[T comparable](t *testing.T, got T, want T) {
	t.Helper()

	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func ptr[T any](value T) *T {
	return &value
}
