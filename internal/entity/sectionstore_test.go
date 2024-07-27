package entity

import (
	"reflect"
	"testing"
)

func TestSectionStore_AssertSize(t *testing.T) {
	type fields struct {
		Sections map[string]*Section
	}
	type args struct {
		size uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "AssertSize works correctly",
			fields: fields{
				Sections: map[string]*Section{
					"section1": {
						FileSize: 10,
					},
				},
			},
			args: args{
				size: 20,
			},
			wantErr: false,
		},
		{
			name: "AssertSize throws error",
			fields: fields{
				Sections: map[string]*Section{
					"section1": {
						FileSize: 10,
					},
					"section2": {
						FileSize: 15,
					},
				},
			},
			args: args{
				size: 20,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				Sections: tt.fields.Sections,
			}
			if err := s.AssertSize(tt.args.size); (err != nil) != tt.wantErr {
				t.Errorf("AssertSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSectionStore_FindSection(t *testing.T) {
	type fields struct {
		Sections map[string]*Section
	}
	type args struct {
		addr uint64
		size uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Section
	}{
		{
			name: "FindSection failed",
			fields: fields{
				Sections: map[string]*Section{
					"section1": {
						Debug:   true,
						Addr:    100,
						AddrEnd: 200,
					},
				},
			},
			args: args{
				addr: 150,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				Sections: tt.fields.Sections,
			}
			if got := s.FindSection(tt.args.addr, tt.args.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindSection() = %v, want %v", got, tt.want)
			}
		})
	}
}
