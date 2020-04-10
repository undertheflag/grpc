package main

import (
	"context"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	pb "grpc/productinfo/client/ecommerce"
	"log"
	"time"
)

const (
	address = "localhost:50051"
	hostname = "localhost"
	crtFile = "E:\\code\\go\\grpc\\certs\\server.crt"
)

func fetchToken() *oauth2.Token {
	return &oauth2.Token{AccessToken: "some-secret-token"}
}

func main() {

	perRPC := oauth.NewOauthAccess(fetchToken())

	creds, err := credentials.NewClientTLSFromFile(crtFile, hostname)
	if err != nil {
		log.Fatalf("failed to load credentials : %v", err)
	}

	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(perRPC),
		// transport credentials.
		grpc.WithTransportCredentials(creds),
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
