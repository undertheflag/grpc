package main

import (
	"context"
	"encoding/base64"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "grpc/productinfo/client/ecommerce"
	"log"
	"time"
)

const (
	address  = "localhost:50051"
	hostname = "localhost"
	crtFile  = "E:\\code\\go\\grpc\\certs\\server.crt"
)

type basicAuth struct {
	username string
	password string
}

func (b basicAuth) GetRequestMetadata(
	ctx context.Context,
	in ...string,
) (map[string]string, error) {
	auth := b.username + ":" + b.password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))
	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

func (b basicAuth) RequireTransportSecurity() bool {
	return true
}

func main() {
	creds, err := credentials.NewClientTLSFromFile(crtFile, hostname)
	if err != nil {
		log.Fatalf("failed to load credentials : %v", err)
	}

	auth := basicAuth{
		username: "admin",
		password: "admin",
	}
	opts := []grpc.DialOption{
		// transport credentials.
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth),
	}

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewProductInfoClient(conn)
	//添加商品
	name := "Apple iPhone 11"
	description := "Meet Apple iPhone 11. All-new dual-camera system with Ultra Wide and Night mode."
	price := float32(699.00)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.AddProduct(ctx, &pb.Product{Name: name, Description: description, Price: price})
	if err != nil {
		log.Fatalf("Could not add product: %v", err)
	}
	log.Printf("Product ID : %s added successfully", r.Value)

	p, err := c.GetProduct(ctx, &pb.ProductID{Value: r.Value})
	if err != nil {
		log.Fatalf("Could not get product : %v", err)
	}
	log.Printf("Product: %v", p.String())
}
