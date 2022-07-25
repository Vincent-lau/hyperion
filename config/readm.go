package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)



func readMat() [][]int {
	file, err := os.Open("data/adj.txt")	
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var m [][]int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		row := make([]int, 0)
		r := strings.Split(scanner.Text(), ",")	
		
		for _, v := range r {		
			i, err := strconv.Atoi(v)
			if err != nil {
				log.Fatal(err)
			}
			row = append(row, i)
		}
		m = append(m, row)
	}


	return m
}


func AdjList() [][]int {
	m := readMat()
	l := make([][]int, len(m))

	for i, r := range(m) {
		for j, c := range(r) {
			if c == 1{
				l[i] = append(l[i], j)
			}
		}
	}

	return l

}

func RAdjList() [][]int {
	m := readMat()
	l := make([][]int, len(m))
	for i, r := range(m) {
		for j, c := range(r) {
			if c == 1 {
				l[j] = append(l[j], i)
			}
		}
	}	

	return l
}
