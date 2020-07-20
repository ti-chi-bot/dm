// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pingcap/dm/dm/worker"
	"github.com/pingcap/dm/pkg/log"
	"github.com/pingcap/dm/pkg/utils"

	"github.com/pingcap/errors"
	"go.uber.org/zap"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg := worker.NewConfig()
	err := cfg.Parse(os.Args[1:])
	switch errors.Cause(err) {
	case nil:
	case flag.ErrHelp:
		os.Exit(0)
	default:
		fmt.Printf("parse cmd flags err: %s", errors.ErrorStack(err))
		os.Exit(2)
	}

	err = log.InitLogger(&log.Config{
		File:   cfg.LogFile,
		Level:  strings.ToLower(cfg.LogLevel),
		Format: cfg.LogFormat,
	})
	if err != nil {
		fmt.Printf("init logger error %v", errors.ErrorStack(err))
		os.Exit(2)
	}

	utils.PrintInfo("dm-worker", func() {
		log.L().Info("", zap.Stringer("dm-worker config", cfg))
	})

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	s := worker.NewServer(cfg)

	go func() {
		sig := <-sc
		log.L().Info("got signal to exit", zap.Stringer("signal", sig))
		s.Close()
	}()

	err = s.Start()
	if err != nil {
		log.L().Error("fail to start dm-worker", zap.Error(err))
	}
	s.Close() // wait until closed
	log.L().Info("dm-worker exit")

	syncErr := log.L().Sync()
	if syncErr != nil {
		fmt.Fprintln(os.Stderr, "sync log failed", syncErr)
	}

	if err != nil || syncErr != nil {
		os.Exit(1)
	}
}
