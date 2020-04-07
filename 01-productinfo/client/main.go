package main

import (
	"context"
	"google.golang.org/grpc"
	pb "grpc/productinfo/client/ecommerce"
	"log"
	"time"
)

const (
	address = "localhost:50051"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
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
