package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"log"
	"time"

	jobq "github.com/NautiluX/gojobqueue"
)

func init() {
	jobq.RegisterJob("MyCustomJob", DecodeCustomJob)
}

func main() {
	jobq.Flags()
	flag.Parse()
	mode := jobq.Start()
	defer jobq.Stop()

	if mode != jobq.ClientMode {
		for i := 0; ; i++ {
			job := CustomJob{1 * time.Second}
			jobq.EnqueueJob("MyCustomJob", job)
			time.Sleep(1 * time.Second)
		}
	}
}

type CustomJob struct {
	Delay time.Duration
}

func DecodeCustomJob(content []byte) jobq.JobType {
	var jobIncoming CustomJob
	b := bytes.Buffer{}
	b.Write(content)
	d := gob.NewDecoder(&b)
	err := d.Decode(&jobIncoming)
	if err != nil {
		log.Fatalf("failed decode job: %v", err)
	}
	return jobIncoming
}

func (j CustomJob) Process() {
	time.Sleep(j.Delay)
}
