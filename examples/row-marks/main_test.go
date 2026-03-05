package main

import (
	"reflect"
	"testing"

	"github.com/iw2rmb/flourish/editor"
)

func TestComputeRowMarks(t *testing.T) {
	tests := []struct {
		name string
		base []string
		cur  []string
		want map[int]editor.RowMarkState
	}{
		{
			name: "delete middle line anchors above next row",
			base: []string{"a", "b", "c"},
			cur:  []string{"a", "c"},
			want: map[int]editor.RowMarkState{
				1: {DeletedAbove: true},
			},
		},
		{
			name: "undo returns to baseline with no marks",
			base: []string{"a", "b", "c"},
			cur:  []string{"a", "b", "c"},
			want: map[int]editor.RowMarkState{},
		},
		{
			name: "insert line marks inserted row",
			base: []string{"a", "c"},
			cur:  []string{"a", "b", "c"},
			want: map[int]editor.RowMarkState{
				1: {Inserted: true},
			},
		},
		{
			name: "replace one line marks updated row",
			base: []string{"a", "b", "c"},
			cur:  []string{"a", "x", "c"},
			want: map[int]editor.RowMarkState{
				1: {Updated: true},
			},
		},
		{
			name: "delete trailing line anchors below last row",
			base: []string{"a", "b", "c"},
			cur:  []string{"a", "b"},
			want: map[int]editor.RowMarkState{
				1: {DeletedBelow: true},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeRowMarks(tc.base, tc.cur)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("marks=%v, want %v", got, tc.want)
			}
		})
	}
}
