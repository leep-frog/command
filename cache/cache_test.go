package cache

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPut(t *testing.T) {
	for _, test := range []struct {
		name    string
		key     string
		data    string
		wantErr string
	}{
		{
			name:    "put fails on empty",
			wantErr: "invalid key format",
		}, {
			name:    "put fails on invalid key",
			key:     "abc-$",
			wantErr: "invalid key format",
		}, {
			name: "put succeeds",
			key:  "abc",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := NewTestCache(t)
			defer os.RemoveAll(c.dir)

			err := c.Put(test.key, test.data)
			if test.wantErr == "" && err != nil {
				t.Errorf("Put(%s, %s) returned err %v; want nil", test.key, test.data, err)
			}
			if test.wantErr != "" && err == nil {
				t.Errorf("Put(%s, %s) returned nil; want err %q", test.key, test.data, test.wantErr)
			}
			if test.wantErr != "" && err != nil && !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("Put(%s, %s) returned err %q; want %q", test.key, test.data, err.Error(), test.wantErr)
			}
		})
	}
}

func TestGet(t *testing.T) {
	for _, test := range []struct {
		name    string
		key     string
		want    string
		wantErr string
	}{
		{
			name:    "get fails on empty",
			wantErr: "invalid key format",
		}, {
			name:    "get fails on invalid key",
			key:     "abc-$",
			wantErr: "invalid key format",
		}, {
			name: "returns empty string on missing key",
			key:  "xyz",
		}, {
			name: "returns value on valid key",
			key:  "abc",
			want: "123\n456\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Temporarily change cache dir
			c := NewTestCache(t)
			defer os.RemoveAll(c.dir)

			if err := c.Put("abc", "123\n456\n"); err != nil {
				t.Fatalf("failed to setup cache: %v", err)
			}

			resp, err := c.Get(test.key)
			if test.wantErr == "" && err != nil {
				t.Errorf("Get(%s) returned err: %v; want nil", test.key, err)
			}
			if test.wantErr != "" && err == nil {
				t.Errorf("Get(%s) returned nil; want err %q", test.key, test.wantErr)
			}
			if test.wantErr != "" && err != nil && !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("Get(%s) returned err: %q; want %q", test.key, err.Error(), test.wantErr)
			}

			if diff := cmp.Diff(test.want, resp); diff != "" {
				t.Fatalf("Get(%s) produced diff :\n%s", test.key, diff)
			}
		})
	}
}

type testStruct struct {
	A int64
	B string
	C map[string]bool
}

func TestPutStruct(t *testing.T) {
	for _, test := range []struct {
		name    string
		key     string
		data    interface{}
		want    string
		wantErr string
	}{
		{
			name:    "put fails on empty",
			wantErr: "invalid key format",
		}, {
			name:    "put fails on invalid key",
			key:     "abc-$",
			wantErr: "invalid key format",
		}, {
			name: "put succeeds",
			key:  "abc",
			data: &testStruct{
				A: 1,
				B: "two",
				C: map[string]bool{
					"three": true,
				},
			},
			want: strings.Join([]string{
				"{",
				`  "A": 1,`,
				`  "B": "two",`,
				`  "C": {`,
				`    "three": true`,
				"  }",
				"}",
			}, "\n"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := NewTestCache(t)
			defer os.RemoveAll(c.dir)

			err := c.PutStruct(test.key, test.data)
			if test.wantErr == "" && err != nil {
				t.Errorf("PutStruct(%s, %v) returned err %v; want nil", test.key, test.data, err)
			}
			if test.wantErr != "" && err == nil {
				t.Errorf("PutStruct(%s, %v) returned nil; want err %q", test.key, test.data, test.wantErr)
			}
			if test.wantErr != "" && err != nil && !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("PutStruct(%s, %v) returned err %q; want %q", test.key, test.data, err.Error(), test.wantErr)
			}

			stored, _ := c.Get(test.key)
			if diff := cmp.Diff(test.want, stored); diff != "" {
				t.Errorf("PutStruct(%s, %v) produced diff:\n%s", test.key, test.data, diff)
			}
		})
	}
}
