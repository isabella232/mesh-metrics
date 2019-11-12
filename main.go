package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daxmc99/prometheus-scraper/srv"
	"github.com/google/uuid"
	"github.com/linkerd/linkerd2/pkg/admin"
	"github.com/linkerd/linkerd2/pkg/flags"
	"github.com/prometheus/client_golang/api"
	"github.com/sirupsen/logrus"
)

func main() {

	cmd := flag.NewFlagSet("public-api", flag.ExitOnError)
	addr := cmd.String("addr", "127.0.0.1:8084", "address to serve on")
	metricsAddr := cmd.String("metrics-addr", "127.0.0.1:9958", "address to serve scrapable metrics on")
	apiAddr := cmd.String("api-addr", "http://linkerd-prometheus.linkerd.svc.cluster.local:9090", "address of the prometheus service")
	prometheusNamespace := cmd.String("controller-namespace", "linkerd", "namespace in which Linkerd is installed")
	debug := cmd.Bool("debug", false, "enable debug logging")
	//kubeConfigPath := cmd.String("kubeconfig", "", "path to kube config")

	flags.ConfigureAndParse(cmd, os.Args[1:])

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	// TODO Sanity check we can connect here and verify promtheus server has correct config

	client, err := api.NewClient(api.Config{
		Address: *apiAddr,
	})
	if err != nil {
		logrus.Fatalf("failed to construct client for API server URL %s", *apiAddr)
	}

	//TODO Figure out a better
	clusterDomain := "cluster.local"

	// k8sAPI, err := k8s.NewAPI(*kubeConfigPath, "", "", 0)
	// if err != nil {
	// 	logrus.Fatalf("failed to construct Kubernetes API client: [%s]", err)
	// }

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	uuidstring := uuid.New().String()

	server := srv.NewServer(*addr, *prometheusNamespace, clusterDomain, true, uuidstring, client)

	go func() {
		logrus.Infof("starting HTTP server on %+v", *addr)
		server.ListenAndServe()
	}()

	go admin.StartServer(*metricsAddr)

	<-stop

	logrus.Infof("shutting down HTTP server on %+v", *addr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
