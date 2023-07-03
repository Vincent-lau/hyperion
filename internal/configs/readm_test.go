package config

import (
	"testing"
)

func TestReadMat(t *testing.T) {

	m := readMat()
	ans := [][]int{
		{0, 0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0, 0},
		{1, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0, 1, 1, 0},
	}

	for i := range m {
		for j := range m[i] {
			if m[i][j] != ans[i][j] {
				t.Fatalf(`m[%d][%d] = %d, want %d`, i, j, m[i][j], ans[i][j])
			}
		}
	}

}

func TestReadAdj(t *testing.T) {
	l := AdjList()
	ans := [][]int {
		{5},
		{3},
		{8},
		{8},
		{6},
		{4},
		{2},
		{0},
		{1, 6, 7},
	}
	for i := range l {
		for j := range l[i] {
			if l[i][j] != ans[i][j] {
				t.Fatalf(`l[%d][%d] = %d, want %d`, i, j, l[i][j], ans[i][j])
			}
		}
	}
}
