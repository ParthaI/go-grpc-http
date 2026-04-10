package order

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderv1 "github.com/parthasarathi/go-grpc-http/gen/go/order/v1"
	"github.com/parthasarathi/go-grpc-http/internal/order/command"
	"github.com/parthasarathi/go-grpc-http/internal/order/model"
	"github.com/parthasarathi/go-grpc-http/internal/order/query"
)

// CommandServer handles write operations (CQRS command side).
type CommandServer struct {
	orderv1.UnimplementedOrderCommandServiceServer
	handler *command.Handler
}

func NewCommandServer(handler *command.Handler) *CommandServer {
	return &CommandServer{handler: handler}
}

func (s *CommandServer) PlaceOrder(ctx context.Context, req *orderv1.PlaceOrderRequest) (*orderv1.PlaceOrderResponse, error) {
	if req.UserId == "" || len(req.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and items are required")
	}

	items := make([]model.OrderItem, len(req.Items))
	for i, item := range req.Items {
		if item.ProductId == "" || item.Quantity <= 0 {
			return nil, status.Error(codes.InvalidArgument, "each item needs product_id and positive quantity")
		}
		items[i] = model.OrderItem{
			ProductID: item.ProductId,
			Quantity:  item.Quantity,
		}
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	order, err := s.handler.PlaceOrder(ctx, req.UserId, items, currency)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient stock") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "place order: %v", err)
	}

	return &orderv1.PlaceOrderResponse{
		OrderId:   order.ID,
		Status:    string(order.Status),
		TotalCents: order.TotalCents,
		CreatedAt: timestamppb.Now(),
	}, nil
}

func (s *CommandServer) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.CancelOrderResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	err := s.handler.CancelOrder(ctx, req.OrderId, req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		if strings.Contains(err.Error(), "can only cancel") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "cancel order: %v", err)
	}

	return &orderv1.CancelOrderResponse{
		OrderId: req.OrderId,
		Status:  string(model.StatusCancelled),
	}, nil
}

// QueryServer handles read operations (CQRS query side).
type QueryServer struct {
	orderv1.UnimplementedOrderQueryServiceServer
	handler *query.Handler
}

func NewQueryServer(handler *query.Handler) *QueryServer {
	return &QueryServer{handler: handler}
}

func (s *QueryServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	view, err := s.handler.GetOrder(ctx, req.OrderId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "get order: %v", err)
	}

	return viewToProto(view), nil
}

func (s *QueryServer) ListOrdersByUser(ctx context.Context, req *orderv1.ListOrdersByUserRequest) (*orderv1.ListOrdersByUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	views, err := s.handler.ListOrdersByUser(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list orders: %v", err)
	}

	resp := &orderv1.ListOrdersByUserResponse{}
	for _, v := range views {
		resp.Orders = append(resp.Orders, viewToProto(v))
	}
	return resp, nil
}

func viewToProto(v *model.OrderView) *orderv1.GetOrderResponse {
	resp := &orderv1.GetOrderResponse{
		OrderId:    v.OrderID,
		UserId:     v.UserID,
		TotalCents: v.TotalCents,
		Currency:   v.Currency,
		Status:     v.Status,
		PaymentId:  v.PaymentID,
		CreatedAt:  timestamppb.New(v.CreatedAt),
		UpdatedAt:  timestamppb.New(v.UpdatedAt),
	}
	for _, item := range v.Items {
		resp.Items = append(resp.Items, &orderv1.OrderItem{
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			PriceCents:  item.PriceCents,
		})
	}
	return resp
}
