package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	restAddr := ":8080"
	grpcAddr := ":50051"

	// Start REST Server
	restServer := StartRESTServer(restAddr)
	go func() {
		log.Printf("Starting REST Server on %s\n", restAddr)
		if err := restServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("REST Server failed: %v\n", err)
		}
	}()

	// Start gRPC Server
	grpcServer, lis, err := StartGRPCServer(grpcAddr)
	if err != nil {
		log.Fatalf("Failed to initialize gRPC listener: %v\n", err)
	}
	go func() {
		log.Printf("Starting gRPC Server on %s\n", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC Server failed: %v\n", err)
		}
	}()

	// Wait for terminate signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down servers...")
	_ = restServer.Close()
	grpcServer.GracefulStop()
	log.Println("Servers stopped.")
}
