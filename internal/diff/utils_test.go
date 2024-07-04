package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func Test_requireAnalyzeModeSame(t *testing.T) {
	type args struct {
		oldResult *commonResult
		newResult *commonResult
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "same order",
			args: args{
				oldResult: &commonResult{
					Analyzers: []entity.Analyzer{"a", "b"},
				},
				newResult: &commonResult{
					Analyzers: []entity.Analyzer{"a", "b"},
				},
			},
			want: true,
		},
		{
			name: "different order",
			args: args{
				oldResult: &commonResult{
					Analyzers: []entity.Analyzer{"a", "b"},
				},
				newResult: &commonResult{
					Analyzers: []entity.Analyzer{"b", "a"},
				},
			},
			want: true,
		},
		{
			name: "different",
			args: args{
				oldResult: &commonResult{
					Analyzers: []entity.Analyzer{"a", "b"},
				},
				newResult: &commonResult{
					Analyzers: []entity.Analyzer{"a", "c"},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, requireAnalyzeModeSame(tt.args.oldResult, tt.args.newResult), "requireAnalyzeModeSame(%v, %v)", tt.args.oldResult, tt.args.newResult)
		})
	}
}
