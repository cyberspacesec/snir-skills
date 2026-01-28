package models

import (
	"reflect"
	"testing"
)

func TestHeaderMap(t *testing.T) {
	tests := []struct {
		name    string
		headers []Header
		want    map[string][]string
	}{
		{
			name: "单个头信息测试",
			headers: []Header{
				{Name: "Content-Type", Value: "application/json"},
			},
			want: map[string][]string{
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "多个不同头信息测试",
			headers: []Header{
				{Name: "Content-Type", Value: "application/json"},
				{Name: "Authorization", Value: "Bearer token123"},
			},
			want: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer token123"},
			},
		},
		{
			name: "相同名称多值头信息测试",
			headers: []Header{
				{Name: "Set-Cookie", Value: "sessionId=123"},
				{Name: "Set-Cookie", Value: "userId=456"},
			},
			want: map[string][]string{
				"Set-Cookie": {"sessionId=123", "userId=456"},
			},
		},
		{
			name:    "空头信息测试",
			headers: []Header{},
			want:    map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				Headers: tt.headers,
			}
			got := r.HeaderMap()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HeaderMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
