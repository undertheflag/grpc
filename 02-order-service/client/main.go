package main

import (
	"context"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	pb "grpc/ordermgt/client/ecommerce"
	"io"
	"log"
	"time"
)

const (
	address = "localhost:50051"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect : %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderManagementClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	order1 := pb.Order{
		Id:          "101",
		Items:       []string{"iPhone XS", "Mac Book Pro"},
		Destination: "San Jose, CA",
		Price:       2300.00,
	}

	res, _ := client.AddOrder(ctx, &order1)
	if res != nil {
		log.Print("AddOrder Response -> ", res.Value)
	}

	retrievedOrder, _ := client.GetOrder(ctx, &wrappers.StringValue{Value: "106"})
	log.Print("GetOrder Response -> : ", retrievedOrder)

	searchStream, _ := client.SearchOrders(ctx, &wrappers.StringValue{Value: "Google"})
	for {
		searchOrder, err := searchStream.Recv()
		if err == io.EOF {
			log.Print("EOF")
			break
		}
		if err != nil {
			log.Print("SearchStream Recv error", err)
			break
		}
		log.Print("Search Result : ", searchOrder)
	}

	updOrder1 := pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00}
	updOrder2 := pb.Order{Id: "103", Items: []string{"Apple Watch S4", "Mac Book Pro", "iPad Pro"}, Destination: "San Jose, CA", Price: 2800.00}
	updOrder3 := pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub", "iPad Mini"}, Destination: "Mountain View, CA", Price: 2200.00}

	updateStream, err := client.UpdateOrders(ctx)
	if err != nil {
		log.Fatalf("%v.UpdateOrders(_) = _, %v", client, err)
	}

	if err := updateStream.Send(&updOrder1); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder1, err)
	}
	if err := updateStream.Send(&updOrder2); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder2, err)
	}
	if err := updateStream.Send(&updOrder3); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder3, err)
	}

	updateRes, err := updateStream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", updateStream, err, nil)
	}
	log.Printf("Update Orders Res : %s", updateRes)

	streamProcOrder, err := client.ProcessOrders(ctx)
	if err != nil {
		log.Fatalf("%v.ProcessOrders(_) = _, %v", client, err)
	}

	if err := streamProcOrder.Send(&wrappers.StringValue{Value: "102"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "102", err)
	}
	if err := streamProcOrder.Send(&wrappers.StringValue{Value: "103"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "103", err)
	}
	if err := streamProcOrder.Send(&wrappers.StringValue{Value: "104"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "104", err)
	}

	channel := make(chan struct{})
	go asyncClientBidirectionalRPC(streamProcOrder, channel)
	time.Sleep(time.Microsecond * 10000)

	if err := streamProcOrder.Send(&wrappers.StringValue{Value: "101"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "101", err)
	}
	if err := streamProcOrder.CloseSend(); err != nil {
		log.Fatal(err)
	}
	<-channel
	log.Print("channel close, client close")
}

func asyncClientBidirectionalRPC(streamProcOrder pb.OrderManagement_ProcessOrdersClient, c chan struct{}) {
	for {
		combinedShipment, errProcOrder := streamProcOrder.Recv()
		if errProcOrder == io.EOF {
			break
		}
		log.Printf("Combined shipment : ", combinedShipment.OrdersList)
	}
	c <- struct{}{}
}
