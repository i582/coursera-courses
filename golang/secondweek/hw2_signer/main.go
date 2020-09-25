package main

import (
	"fmt"
)

func main() {
	// var res string

	inputData := []int{0, 1}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				fmt.Println(fibNum)
				out <- fibNum
			}
			// close(out)
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			data, ok := dataRaw.(string)
			fmt.Println(data)
			if !ok {
				panic("cant convert result data to string")
			}
			// res = data
		}),
	}


	ExecutePipeline(hashSignJobs...)

	// time.Sleep(50 * time.Second)
}
