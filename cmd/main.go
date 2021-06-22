package main

import (
	"context"
	"cql-proxy/proxy"
	"cql-proxy/proxycore"
	"fmt"
	"github.com/datastax/go-cassandra-native-protocol/primitive"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func cancelExample() {
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	wg.Add(4)

	for i := 0; i < 4; i++ {
		go func() {
			done := false
			for !done {
				fmt.Println("looping...")
				select {
				case <-ctx.Done():
					err := ctx.Err()
					if err == context.Canceled {
						log.Println("cancelled")
					} else {
						log.Printf("error: %s\n", err)
					}
					done = true
				}
			}
			wg.Done()
		}()
	}

	time.Sleep(5 * time.Second)

	cancel()

	wg.Wait()
}

func closeExample() {
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		listener, err := net.Listen("tcp", "127.0.0.1:8000")
		if err != nil {
			log.Fatalf("unable to listen %v", err)
		}

		wg.Done()

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("unable to accept new connection %v", err)
			}

			go func(c net.Conn) {
				timer := time.NewTimer(2 * time.Second)
				done := false
				for !done {
					select {
					case <-timer.C:
						done = true
					default:
						c.Write([]byte("a"))
					}
				}
				log.Println("closing connection")
				c.Close()
			}(conn)
		}

	}()

	wg.Wait()

	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalf("unable to connect %v", err)
	}

	for {
		b := make([]byte, 16)
		_, err := conn.Read(b)
		if err == net.ErrClosed || err == io.EOF {
			log.Println("closed")
			break
		} else if err != nil {
			log.Printf("error reading %v\n", err)
		}
		//log.Println(string(b[:n]))
	}
}

func singleChannelExample() {

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	type Event struct {
		t string
	}

	ch := make(chan Event)

	wg.Add(4)

	for i := 0; i < 4; i++ {
		go func(i int) {
			done := false
			for !done {
				fmt.Println("looping...")
				select {
				case evt := <-ch:
					fmt.Printf("In goroutine %d: %v\n", i, evt)
				case <-ctx.Done():
					err := ctx.Err()
					if err == context.Canceled {
						log.Println("cancelled")
					} else {
						log.Printf("error: %s\n", err)
					}
					done = true
				}
			}
			wg.Done()
		}(i + 1)
	}

	for i := 0; i < 100; i++ {
		ch <- Event{t: "schema"}
	}

	time.Sleep(1 * time.Second)

	cancel()

	wg.Wait()
}

type ReceiverFunc func(io.Reader) error

func (f ReceiverFunc) Receive(r io.Reader) error {
	return f(r)
}

type SenderFunc func(io.Writer) error

func (f SenderFunc) Send(w io.Writer) error {
	return f(w)
}

func (f SenderFunc) Closing(err error) {
	log.Printf("################################### Closing %v ##################################", err)
}

func connExample() {
	conn, err := proxy.Connect("tcp", "127.0.0.1:8000", ReceiverFunc(func(reader io.Reader) error {
		b := make([]byte, 16)
		n, err := reader.Read(b)
		log.Println(string(b[:n]))
		return err
	}))

	if err != nil {
		log.Fatalf("error connecting  %v", err)
	}

	timer := time.NewTimer(time.Second)

	closed := false
	for !closed {
		select {
		case <-conn.IsClosed():
			log.Printf("closed %v", conn.Err())
			closed = true
		case <-timer.C:
			//conn.Close()
		default:
			conn.Write(SenderFunc(func(writer io.Writer) error {
				_, err := writer.Write([]byte("a"))
				return err
			}))
		}
	}

	time.Sleep(1 * time.Second)
}

func connWithBundleEx() {
	bundle, err := proxycore.LoadBundleZip("secure-connect-testdb1.zip")
	if err != nil {
		log.Fatalf("unable to open bundle: %v", err)
	}

	factory, err := proxycore.ResolveAstra(bundle)
	if err != nil {
		log.Fatalf("unable to resolve endpoints: %v", err)
	}

	ctx := context.Background()

	conn, err := proxycore.ClusterConnect(ctx, factory.ContactPoints()[0])
	if err != nil {
		log.Fatalf("unable to connect to cluster: %v", err)
	}

	auth := proxycore.NewDefaultAuth("HYhtHNEYMKOFpFGyOsAYyHSK", "rEPtSneDWH3Of8HCMQD1d8uANl5.T5NavwIvJLLUivOJsA7fyl9z_4uTNCmHMkgiWcPTz2nCI5,p+3X41hEpdj5fDz,tOa,vjEMmd0K,2wllbPn_dqRZPox5TbP1H,QE")
	version, err := conn.Handshake(ctx, primitive.ProtocolVersion4, auth)
	if err != nil {
		log.Fatalf("unable to connect to cluster: %v", err)
	}
	_ = version

	timer := time.NewTimer(time.Second)

	closed := false
	for !closed {
		select {
		case <-conn.IsClosed():
			log.Printf("closed %v", conn.Err())
			closed = true
		case <-timer.C:
			conn.Close()
		}
	}

	time.Sleep(2 * time.Second)
}

func connClusterWithBundleEx() {
	bundle, err := proxycore.LoadBundleZip("secure-connect-testdb1.zip")
	if err != nil {
		log.Fatalf("unable to open bundle: %v", err)
	}

	factory, err := proxycore.ResolveAstra(bundle)
	if err != nil {
		log.Fatalf("unable to resolve astra: %v", err)
	}

	ctx := context.Background()

	auth := proxycore.NewDefaultAuth("HYhtHNEYMKOFpFGyOsAYyHSK", "rEPtSneDWH3Of8HCMQD1d8uANl5.T5NavwIvJLLUivOJsA7fyl9z_4uTNCmHMkgiWcPTz2nCI5,p+3X41hEpdj5fDz,tOa,vjEMmd0K,2wllbPn_dqRZPox5TbP1H,QE")
	conn, err := proxycore.ConnectToCluster(ctx, primitive.ProtocolVersion4, auth, factory)
	if err != nil {
		log.Fatalf("unable to connect to cluster: %v", err)
	}

	timer := time.NewTimer(time.Second)
	timer.Stop()

	closed := false
	for !closed {
		select {
		case <-conn.IsClosed():
			closed = true
		case <-timer.C:
			conn.Close()
		}
	}

	time.Sleep(2 * time.Second)
}

func main() {
	connClusterWithBundleEx()
}