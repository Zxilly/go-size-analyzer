package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetUrlFromListen(t *testing.T) {
	type args struct {
		listen string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Valid listen address with port",
			args: args{
				listen: "127.0.0.1:8080",
			},
			want: "http://localhost:8080",
		},
		{
			name: "Valid listen address with different port",
			args: args{
				listen: "127.0.0.1:3000",
			},
			want: "http://localhost:3000",
		},
		{
			name: "Listen address without port",
			args: args{
				listen: "127.0.0.1",
			},
			want: "http://localhost:8080",
		},
		{
			name: "Empty listen address",
			args: args{
				listen: "",
			},
			want: "http://localhost:8080",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetUrlFromListen(tt.args.listen), "GetUrlFromListen(%v)", tt.args.listen)
		})
	}
}
