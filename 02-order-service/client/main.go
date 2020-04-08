package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	pb "grpc/ordermgt/client/ecommerce"
	"log"
	"time"
)

const (
	address = "localhost:50051"
)

func main() {
	conn, err := grpc.Dial(
		address,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(orderUnaryClientInterceptor),
		grpc.WithStreamInterceptor(clientStreamInterceptor))
	if err != nil {
		log.Fatalf("did not connect : %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderManagementClient(conn)
	clientDeadline := time.Now().Add(2*time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), clientDeadline)
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	//add order
	order1 := pb.Order{
		Id:          "101",
		Items:       []string{"iPhone XS", "Mac Book Pro"},
		Destination: "San Jose, CA",
		Price:       2300.00,
	}

	res, err := client.AddOrder(ctx, &order1)
	if err != nil {
		got := status.Code(err)
		log.Printf("Error Occured -> addOrder : , %v", got)
	} else {
		log.Print("AddOrder Response -> ", res.Value)
	}

//	retrievedOrder, _ := client.GetOrder(ctx, &wrappers.StringValue{Value: "106"})
//	log.Print("GetOrder Response -> : ", retrievedOrder)
//
//	searchStream, _ := client.SearchOrders(ctx, &wrappers.StringValue{Value: "Google"})
//	for {
//		searchOrder, err := searchStream.Recv()
//		if err == io.EOF {
//			log.Print("EOF")
//			break
//		}
//		if err != nil {
//			log.Print("SearchStream Recv error", err)
//			break
//		}
//		log.Print("Search Result : ", searchOrder)
//	}
//
//	updOrder1 := pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00}
//	updOrder2 := pb.Order{Id: "103", Items: []string{"Apple Watch S4", "Mac Book Pro", "iPad Pro"}, Destination: "San Jose, CA", Price: 2800.00}
//	updOrder3 := pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub", "iPad Mini"}, Destination: "Mountain View, CA", Price: 2200.00}
//
//	updateStream, err := client.UpdateOrders(ctx)
//	if err != nil {
//		log.Fatalf("%v.UpdateOrders(_) = _, %v", client, err)
//	}
//
//	if err := updateStream.Send(&updOrder1); err != nil {
//		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder1, err)
//	}
//	if err := updateStream.Send(&updOrder2); err != nil {
//		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder2, err)
//	}
//	if err := updateStream.Send(&updOrder3); err != nil {
//		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder3, err)
//	}
//
//	updateRes, err := updateStream.CloseAndRecv()
//	if err != nil {
//		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", updateStream, err, nil)
//	}
//	log.Printf("Update Orders Res : %s", updateRes)
//
//	streamProcOrder, err := client.ProcessOrders(ctx)
//	if err != nil {
//		log.Fatalf("%v.ProcessOrders(_) = _, %v", client, err)
//	}
//
//	if err := streamProcOrder.Send(&wrappers.StringValue{Value: "102"}); err != nil {
//		log.Fatalf("%v.Send(%v) = %v", client, "102", err)
//	}
//	if err := streamProcOrder.Send(&wrappers.StringValue{Value: "103"}); err != nil {
//		log.Fatalf("%v.Send(%v) = %v", client, "103", err)
//	}
//	if err := streamProcOrder.Send(&wrappers.StringValue{Value: "104"}); err != nil {
//		log.Fatalf("%v.Send(%v) = %v", client, "104", err)
//	}
//
//	channel := make(chan struct{})
//	go asyncClientBidirectionalRPC(streamProcOrder, channel)
//	time.Sleep(time.Microsecond * 10000)
//
//	if err := streamProcOrder.Send(&wrappers.StringValue{Value: "101"}); err != nil {
//		log.Fatalf("%v.Send(%v) = %v", client, "101", err)
//	}
//	if err := streamProcOrder.CloseSend(); err != nil {
//		log.Fatal(err)
//	}
//	<-channel
//	log.Print("channel close, client close")
}
//
//func asyncClientBidirectionalRPC(streamProcOrder pb.OrderManagement_ProcessOrdersClient, c chan struct{}) {
//	for {
//		combinedShipment, errProcOrder := streamProcOrder.Recv()
//		if errProcOrder == io.EOF {
//			break
//		}
//		log.Printf("Combined shipment : ", combinedShipment.OrdersList)
//	}
//	c <- struct{}{}
//}

func orderUnaryClientInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
	) error {
	//前置
	log.Println("Method : " + method)

	//调用远程方法
	err := invoker(ctx, method, req, reply, cc, opts...)

	//后置
	log.Println("Reply : ", reply)

	return err
}

type wrappedStream struct {
	grpc.ClientStream
}

func newWrappedStream(s grpc.ClientStream) grpc.ClientStream {
	return &wrappedStream{s}
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("====== [Client Stream Interceptor] Receive a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ClientStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	log.Printf("====== [Client Stream Interceptor] Send a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ClientStream.SendMsg(m)
}

func clientStreamInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {

	log.Println("======= [Client Interceptor] ", method)

	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}

	return newWrappedStream(s), nil
}