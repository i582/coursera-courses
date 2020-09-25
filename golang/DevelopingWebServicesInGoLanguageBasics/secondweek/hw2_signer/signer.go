package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	md5Mutex = sync.Mutex{}
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	var outs = make([]chan interface{}, len(jobs)+1)
	for i := range outs {
		outs[i] = make(chan interface{}, 100)
	}
	for i, j := range jobs {
		wg.Add(1)
		go func(index int, j job) {
			j(outs[index], outs[index+1])
			close(outs[index+1])
			wg.Done()
		}(i, j)
	}
	wg.Wait()
}

type crc32res struct {
	id   int64
	hash string
}

func Crc32Worker(id int64, data string, out chan interface{}, wg *sync.WaitGroup) {
	res := DataSignerCrc32(data)
	out <- crc32res{
		id:   id,
		hash: res,
	}
	wg.Done()
}

func SingleHashWorker(data string, out chan interface{}, wg *sync.WaitGroup) {
	md5Mutex.Lock()
	md5 := DataSignerMd5(data)
	md5Mutex.Unlock()
	hashCh := make(chan interface{}, 2)
	wgCrc := &sync.WaitGroup{}
	wgCrc.Add(2)
	go Crc32Worker(0, data, hashCh, wgCrc)
	go Crc32Worker(1, md5, hashCh, wgCrc)
	wgCrc.Wait()
	first := (<-hashCh).(crc32res)
	second := (<-hashCh).(crc32res)
	var res string
	if first.id < second.id {
		res = first.hash + "~" + second.hash
	} else {
		res = second.hash + "~" + first.hash
	}
	out <- res
	wg.Done()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for rawData := range in {
		wg.Add(1)
		data := fmt.Sprint(rawData.(int))
		go SingleHashWorker(data, out, wg)
	}
	wg.Wait()
}

func MultiHashWorker(data string, out chan interface{}, wg *sync.WaitGroup) {
	wgCrc := &sync.WaitGroup{}
	hashCh := make(chan interface{}, 6)
	wgCrc.Add(6)
	for i := 0; i < 6; i++ {
		go Crc32Worker(int64(i), fmt.Sprint(i) + data, hashCh, wgCrc)
	}
	wgCrc.Wait()
	crcRes := make([]crc32res, 0, 6)
	for i := 0; i < 6; i++ {
		crcRes = append(crcRes, (<-hashCh).(crc32res))
	}
	sort.Slice(crcRes, func(i, j int) bool {
		return crcRes[i].id < crcRes[j].id
	})
	var res string
	for _, r := range crcRes {
		res += r.hash
	}
	out <- res
	wg.Done()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for rawData := range in {
		wg.Add(1)
		data := rawData.(string)
		go MultiHashWorker(data, out, wg)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	hashes := make([]string, 0, 100)
	for rawData := range in {
		data := rawData.(string)
		hashes = append(hashes, data)
	}
	sort.Slice(hashes, func(i, j int) bool {
		return hashes[i] < hashes[j]
	})
	res := strings.Join(hashes, "_")
	out <- res
}
