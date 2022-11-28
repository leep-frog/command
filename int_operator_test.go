package command

import (
	"fmt"
	"testing"
)

func TestParseInt(t *testing.T) {
	errFunc := func(s string) error {
		return fmt.Errorf(`strconv.Atoi: parsing %q: invalid syntax`, s)
	}
	for _, test := range []struct {
		name    string
		arg     string
		wantErr error
		want    int
	}{
		{
			name: "No underscore",
			arg:  "123",
			want: 123,
		},
		{
			name:    "Decimal",
			arg:     "12.3",
			wantErr: errFunc("12.3"),
		},
		{
			name: "Underscore in the middle",
			arg:  "1_2_3",
			want: 123,
		},
		{
			name:    "Fails if underscore at the beginning",
			arg:     "_123",
			wantErr: errFunc("_123"),
		},
		{
			name:    "Fails if underscore at the end",
			arg:     "123_",
			wantErr: errFunc("123_"),
		},
		{
			name:    "Fails if multiple underscores in a row",
			arg:     "1__23",
			wantErr: errFunc("1__23"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Single arg
			argName := "i"
			var wd *Data
			if test.wantErr == nil {
				wd = &Data{Values: map[string]interface{}{
					argName: test.want,
				}}
			}
			var stderr string
			if test.wantErr != nil {
				stderr = fmt.Sprintf("%s\n", test.wantErr.Error())
			}
			ExecuteTest(t, &ExecuteTestCase{
				Node:       SerialNodes(Arg[int](argName, testDesc)),
				Args:       []string{test.arg},
				WantErr:    test.wantErr,
				WantStderr: stderr,
				WantData:   wd,
			})

			// List arg
			listArgName := "il"
			wd = nil
			if test.wantErr == nil {
				wd = &Data{Values: map[string]interface{}{
					listArgName: []int{test.want},
				}}
			}
			ExecuteTest(t, &ExecuteTestCase{
				Node:       SerialNodes(ListArg[int](listArgName, testDesc, 1, 3)),
				Args:       []string{test.arg},
				WantErr:    test.wantErr,
				WantStderr: stderr,
				WantData:   wd,
			})
		})
	}
}

func TestParseFloat(t *testing.T) {
	errFunc := func(s string) error {
		return fmt.Errorf(`strconv.ParseFloat: parsing %q: invalid syntax`, s)
	}
	for _, test := range []struct {
		name    string
		arg     string
		wantErr error
		want    float64
	}{
		{
			name: "No underscore",
			arg:  "123",
			want: 123,
		},
		{
			name: "No numbers after decimal",
			arg:  "123.",
			want: 123,
		},
		{
			name: "No underscore with decimals",
			arg:  "123.45",
			want: 123.45,
		},
		{
			name: "No underscore with decimal zero",
			arg:  "123.0",
			want: 123,
		},
		{
			name: "No underscore with multiple decimal zeros",
			arg:  "123.000",
			want: 123,
		},
		{
			name: "No underscore with multiple trailing decimal zeros",
			arg:  "123.0300",
			want: 123.03,
		},
		{
			name: "Underscore in the middle",
			arg:  "1_2_3",
			want: 123,
		},
		{
			name: "Underscore in the middle of decimal",
			arg:  "1_2_3.4_5_6",
			want: 123.456,
		},
		{
			name:    "Fails if underscore at the beginning",
			arg:     "_123",
			wantErr: errFunc("_123"),
		},
		{
			name:    "Fails if underscore at the beginning of decimal",
			arg:     "123._4",
			wantErr: errFunc("123._4"),
		},
		{
			name:    "Fails if underscore at the end",
			arg:     "123_",
			wantErr: errFunc("123_"),
		},
		{
			name:    "Fails if underscore at the end of decimal",
			arg:     "123.456_",
			wantErr: errFunc("123.456_"),
		},
		{
			name:    "Fails if multiple underscores in a row",
			arg:     "1__23",
			wantErr: errFunc("1__23"),
		},
		{
			name:    "Fails if multiple underscores in a row in decimal",
			arg:     "123.45__6",
			wantErr: errFunc("123.45__6"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Single arg
			argName := "f"
			var wd *Data
			if test.wantErr == nil {
				wd = &Data{Values: map[string]interface{}{
					argName: test.want,
				}}
			}
			var stderr string
			if test.wantErr != nil {
				stderr = fmt.Sprintf("%s\n", test.wantErr.Error())
			}
			ExecuteTest(t, &ExecuteTestCase{
				Node:       SerialNodes(Arg[float64](argName, testDesc)),
				Args:       []string{test.arg},
				WantErr:    test.wantErr,
				WantStderr: stderr,
				WantData:   wd,
			})

			// List arg
			listArgName := "fl"
			wd = nil
			if test.wantErr == nil {
				wd = &Data{Values: map[string]interface{}{
					listArgName: []float64{test.want},
				}}
			}
			ExecuteTest(t, &ExecuteTestCase{
				Node:       SerialNodes(ListArg[float64](listArgName, testDesc, 1, 3)),
				Args:       []string{test.arg},
				WantErr:    test.wantErr,
				WantStderr: stderr,
				WantData:   wd,
			})
		})
	}
}
