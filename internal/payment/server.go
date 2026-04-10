package payment

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentv1 "github.com/parthasarathi/go-grpc-http/gen/go/payment/v1"
	"github.com/parthasarathi/go-grpc-http/internal/payment/model"
	"github.com/parthasarathi/go-grpc-http/internal/payment/repository"
)

type Server struct {
	paymentv1.UnimplementedPaymentServiceServer
	repo *repository.Repository
}

func NewServer(repo *repository.Repository) *Server {
	return &Server{repo: repo}
}

func (s *Server) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	if req.PaymentId == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_id is required")
	}

	p, err := s.repo.GetByID(ctx, req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get payment: %v", err)
	}
	if p == nil {
		return nil, status.Error(codes.NotFound, "payment not found")
	}

	return paymentToProto(p), nil
}

func (s *Server) ListPaymentsByOrder(ctx context.Context, req *paymentv1.ListPaymentsByOrderRequest) (*paymentv1.ListPaymentsByOrderResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	payments, err := s.repo.GetByOrderID(ctx, req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list payments: %v", err)
	}

	resp := &paymentv1.ListPaymentsByOrderResponse{}
	for _, p := range payments {
		resp.Payments = append(resp.Payments, paymentToProto(p))
	}
	return resp, nil
}

func (s *Server) RefundPayment(ctx context.Context, req *paymentv1.RefundPaymentRequest) (*paymentv1.RefundPaymentResponse, error) {
	if req.PaymentId == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_id is required")
	}

	p, err := s.repo.GetByID(ctx, req.PaymentId)
	if err != nil || p == nil {
		return nil, status.Error(codes.NotFound, "payment not found")
	}

	if p.Status != model.StatusCompleted {
		return nil, status.Error(codes.FailedPrecondition,
			"can only refund completed payments, current: "+string(p.Status))
	}

	if err := s.repo.UpdateStatus(ctx, req.PaymentId, model.StatusRefunded); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "payment not found")
		}
		return nil, status.Errorf(codes.Internal, "refund: %v", err)
	}

	return &paymentv1.RefundPaymentResponse{
		PaymentId: req.PaymentId,
		Status:    string(model.StatusRefunded),
	}, nil
}

func paymentToProto(p *model.Payment) *paymentv1.GetPaymentResponse {
	return &paymentv1.GetPaymentResponse{
		PaymentId:   p.ID,
		OrderId:     p.OrderID,
		AmountCents: p.AmountCents,
		Currency:    p.Currency,
		Status:      string(p.Status),
		Method:      p.Method,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}
