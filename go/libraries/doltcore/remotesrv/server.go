// Copyright 2019 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remotesrv

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	remotesapi "github.com/dolthub/dolt/go/gen/proto/dolt/services/remotesapi/v1alpha1"
	"github.com/dolthub/dolt/go/libraries/utils/filesys"
)

type Server struct {
	wg       sync.WaitGroup
	stopChan chan struct{}

	grpcPort int
	grpcSrv  *grpc.Server
	httpPort int
	httpSrv  http.Server
}

func (s *Server) GracefulStop() {
	close(s.stopChan)
	s.wg.Wait()
}

func NewServer(httpHost string, httpPort, grpcPort int, fs filesys.Filesys, dbCache DBCache, readOnly bool) *Server {
	s := new(Server)
	s.stopChan = make(chan struct{})

	expectedFiles := newFileDetails()

	s.wg.Add(2)
	s.grpcPort = grpcPort
	s.grpcSrv = grpc.NewServer(grpc.MaxRecvMsgSize(128 * 1024 * 1024))
	var chnkSt remotesapi.ChunkStoreServiceServer = NewHttpFSBackedChunkStore(httpHost, dbCache, expectedFiles, fs)
	if readOnly {
		chnkSt = ReadOnlyChunkStore{chnkSt}
	}
	remotesapi.RegisterChunkStoreServiceServer(s.grpcSrv, chnkSt)

	var handler http.Handler = newFileHandler(dbCache, expectedFiles, fs, readOnly)
	if httpPort == grpcPort {
		handler = grpcMultiplexHandler(s.grpcSrv, handler)
	} else {
		s.wg.Add(2)
	}

	s.httpPort = httpPort
	s.httpSrv = http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: handler,
	}

	return s
}

func grpcMultiplexHandler(grpcSrv *grpc.Server, handler http.Handler) http.Handler {
	h2s := &http2.Server{}
	newHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcSrv.ServeHTTP(w, r)
		} else {
			handler.ServeHTTP(w, r)
		}
	})
	return h2c.NewHandler(newHandler, h2s)
}

func (s *Server) Serve() error {
	httpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.httpPort))
	if err != nil {
		return err
	}

	if s.grpcPort != s.httpPort {
		grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.grpcPort))
		if err != nil {
			httpListener.Close()
			return err
		}

		go func() {
			defer s.wg.Done()
			log.Println("Starting grpc server on port", s.grpcPort)
			err = s.grpcSrv.Serve(grpcListener)
			log.Println("grpc server exited. error:", err)
		}()
		go func() {
			defer s.wg.Done()
			<-s.stopChan
			s.grpcSrv.GracefulStop()
		}()
	}

	go func() {
		defer s.wg.Done()
		log.Println("Starting http server on port ", s.httpPort)
		err := s.httpSrv.Serve(httpListener)
		log.Println("http server exited. exit error:", err)
	}()
	go func() {
		defer s.wg.Done()
		<-s.stopChan
		s.httpSrv.Shutdown(context.Background())
	}()

	s.wg.Wait()
	return nil
}