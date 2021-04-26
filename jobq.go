package jobq

import (
	"bytes"
	"context"
	"encoding/gob"
	"flag"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"time"
)

type runmode string

const (
	SoloMode   runmode = "solo"
	ServerMode runmode = "server"
	ClientMode runmode = "client"
)

func SelectMode(input string) runmode {
	switch input {
	case string(ClientMode):
		return ClientMode
	case string(ServerMode):
		return ServerMode
	default:
		return SoloMode
	}
}

var mode *string
var worker *int
var server *string

func Flags() {
	mode = flag.String("mode", "solo", "Set run mode. Supported modes: solo, server, client")
	worker = flag.Int("worker", 3, "Set worker count.")
	server = flag.String("server", "127.0.0.1:1234", "Set server IP. Only suitable for client and server mode")
}

var jobChan chan Job = make(chan Job, 100)

func Start() runmode {

	runMode := SelectMode(*mode)
	log.Printf("Running in %v mode", runMode)

	// make a channel with a capacity of 100.
	switch runMode {
	case SoloMode:
		solo(*worker)
	case ServerMode:
		startServer(*server)
	case ClientMode:
		startClient(*server, *worker)
	}
	return runMode
}

func Stop() {
	wg.Wait()
}

type Job struct {
	Id         int
	JobType    string
	JobContent []byte
}

type JobType interface {
	Process()
}

var wg sync.WaitGroup

type Worker struct {
	WorkerId int
}

func startWorker(id int, jobChan <-chan Job) {
	defer wg.Done()
	w := Worker{id}

	for job := range jobChan {
		w.processJob(job)
	}
}

func (w *Worker) processJob(job Job) {
	log.Printf("Worker %d processing job %d of type %s.", w.WorkerId, job.Id, job.JobType)
	decode, ok := jobDecoders[job.JobType]
	if ok {
		jobImpl := decode(job.JobContent)
		jobImpl.Process()
		log.Printf("Worker %d finished job %d.", w.WorkerId, job.Id)
		return
	}
	log.Fatalf("Can't create job of type %s", job.JobType)
}

func solo(workerCount int) {

	// start the worker
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go startWorker(i, jobChan)
	}

}

type Server struct {
	queue []Job
	mu    sync.Mutex
}

type RpcArgs struct {
}

func (s *Server) GetJob(args *RpcArgs, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) > 0 {
		*job = s.queue[0]
		s.queue = s.queue[1:]
		return nil
	}

	*job = Job{-1, "", []byte{}}
	return nil
}

func (s *Server) buildQueue(jobChan chan Job) {
	for job := range jobChan {
		s.queue = append(s.queue, job)
		log.Printf("Current queue length: %d", len(s.queue))
	}
}

func startServer(server string) {
	s := new(Server)
	err := rpc.Register(s)
	if err != nil {
		log.Fatal("Cannot register RPC:", err)
	}
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", server)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	go http.Serve(l, nil)

	go s.buildQueue(jobChan)
	log.Println("Server started")

}

func remoteWorker(ctx context.Context, id int, client *rpc.Client) {
	defer wg.Done()
	worker := Worker{id}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping worker %d", id)
			return
		default:
			job := Job{}
			err := client.Call("Server.GetJob", &RpcArgs{}, &job)
			if err != nil {
				log.Println("Failed to get work", err)
			}

			if job.Id != -1 {
				worker.processJob(job)
				continue
			}
			log.Printf("Nothing to do, delaying for 1 second")
			time.Sleep(1 * time.Second)
		}
	}

}

func startClient(server string, workerCount int) {
	client, err := rpc.DialHTTP("tcp", server)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// start the worker
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go remoteWorker(ctx, i, client)
	}

	// Run until ctrl-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	sig := <-c
	log.Printf("Got signal %v", sig)
	cancel()
	wg.Wait()
}

type JobDecoder func([]byte) JobType

var jobDecoders map[string]JobDecoder = make(map[string]JobDecoder)

func RegisterJob(jobName string, decoder JobDecoder) {
	jobDecoders[jobName] = decoder
	log.Printf("registered %s", jobName)
}

var jobCount int

func EnqueueJob(jobType string, job JobType) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(job)
	if err != nil {
		log.Fatalf("failed encode job: %v", err)
	}
	jobChan <- Job{jobCount, jobType, b.Bytes()}
	jobCount++
}
