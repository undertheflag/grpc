package main

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	pb "grpc/ordermgt/server/ecommerce"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const (
	port           = ":50051"
	orderBatchSize = 3
)

var orderMap = make(map[string]pb.Order)

type server struct {
	orderMap map[string]*pb.Order //命名暴露了底层实现，直接用orders更好
}

func (s *server) AddOrder(ctx context.Context, o *pb.Order) (*wrappers.StringValue, error) {
	log.Printf("Order Added. ID : %v", o.Id)
	orderMap[o.Id] = *o

	//******** Reading Metadata from Client ******
	md, metadataAvailable := metadata.FromIncomingContext(ctx)
	if !metadataAvailable {
		return nil, status.Errorf(codes.DataLoss, "UnaryEcho: failed to get metadata")
	}
	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf("====> Metadata %d. %s\n", i, e)
		}
	}

	// creating and sending a header
	header := metadata.New(map[string]string{"location": "San Jose", "timestamp": time.Now().Format(time.StampNano)})
	grpc.SendHeader(ctx, header)

	return &wrappers.StringValue{Value: "Order Added: " + o.Id}, nil
}

func (s *server) GetOrder(ctx context.Context, orderId *wrappers.StringValue) (*pb.Order, error) {
	ord, exists := orderMap[orderId.Value]
	if exists {
		return &ord, status.New(codes.OK, "").Err()
	}

	return nil, status.Errorf(codes.NotFound, "Order does not exist. : ", orderId)
}

func (s *server) SearchOrders(searchQuery *wrappers.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {
	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
		stream.SetTrailer(trailer)
	}()

	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.StampNano)})
	stream.SendHeader(header)

	for key, order := range orderMap {
		log.Print(key, order)

		for _, itemStr := range order.Items {
			log.Print(itemStr)

			if strings.Contains(itemStr, searchQuery.Value) {
				err := stream.Send(&order)
				if err != nil {
					return fmt.Errorf("error sending message to stream : %v ", err)
				}
				log.Print("Matching Order Found : " + key)
				break
			}
		}
	}

	return nil
}

func (s *server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {
	ordersStr := "Updated Order IDs : "
	for {
		order, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&wrappers.StringValue{Value: "Orders processed " + ordersStr})
		}
		if err != nil {
			return err
		}

		orderMap[order.Id] = *order
		log.Printf("Order ID : %s - %s", order.Id, "Updated")
		ordersStr += order.Id + ", "
	}
}

func (s server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {
	batchMarker := 1
	var combinedShipmentMap = make(map[string]pb.CombinedShipment) // 地址->发货订单记录

	for {
		orderId, err := stream.Recv()
		log.Printf("reading Proc order : %s ", orderId)
		if err == io.EOF {
			log.Printf("EOF : %s", orderId)
			for _, shipment := range combinedShipmentMap {
				if err := stream.Send(&shipment); err != nil {
					return err
				}
			}
			return nil
		}
		if err != nil {
			log.Println(err)
			return err
		}

		destination := orderMap[orderId.GetValue()].Destination //获取订单收货地址
		//该地址是否已经有发货记录
		shipment, found := combinedShipmentMap[destination]
		if found {
			ord := orderMap[orderId.GetValue()]
			shipment.OrdersList = append(shipment.OrdersList, &ord)
			combinedShipmentMap[destination] = shipment
		} else {
			comShip := pb.CombinedShipment{
				Id:     "cmb - " + (orderMap[orderId.GetValue()].Destination),
				Status: "Processed!",
			}
			ord := orderMap[orderId.GetValue()]
			comShip.OrdersList = append(shipment.OrdersList, &ord)
			combinedShipmentMap[destination] = comShip
			log.Print(len(comShip.OrdersList), comShip.GetId())
		}

		if batchMarker == orderBatchSize {
			for _, comb := range combinedShipmentMap {
				log.Printf("Shipping : %v -> %v", comb.Id, len(comb.OrdersList))
				if err := stream.Send(&comb); err != nil {
					return err
				}
			}
			batchMarker = 0
			combinedShipmentMap = make(map[string]pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}
}

func orderUnaryServerInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	log.Println("====== [Server Interceptor] ", info.FullMethod)
	log.Printf("Pre Proc Message : %s", req)

	m, err := handler(ctx, req)

	log.Println("====== [Server Interceptor] : postprocessing")
	return m, err
}

// wrappedStream wraps around the embedded grpc.ServerStream, and intercepts the RecvMsg and
// SendMsg method call.
type wrappedStream struct {
	grpc.ServerStream
}

func newWrappedStream(s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s}
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("====== [Server Stream Interceptor Wrapper] Receive a message (Type: %T) at %s", m, time.Now().Format(time.RFC3339))
	return w.ServerStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	log.Printf("====== [Server Stream Interceptor Wrapper] Send a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ServerStream.SendMsg(m)
}

func orderServerStreamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
	) error {

	log.Println("====== [Server Stream Interceptor] ", info.FullMethod)
	err := handler(srv, newWrappedStream(ss))

	if err != nil {
		log.Printf("RPC failed with error : %v", err)
	}
	log.Println("post stream interceptor")
	return err
}

func main() {
	initSampleData()

	s := grpc.NewServer(
		grpc.UnaryInterceptor(orderUnaryServerInterceptor),
		grpc.StreamInterceptor(orderServerStreamInterceptor),
		) //构造gRPC服务对象

	pb.RegisterOrderManagementServer(s, &server{}) //注册服务

	//监听端口,提供gRPC服务
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen %v", err)
	}
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func initSampleData() {
	orderMap["102"] = pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	orderMap["103"] = pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	orderMap["104"] = pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	orderMap["105"] = pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	orderMap["106"] = pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
}
