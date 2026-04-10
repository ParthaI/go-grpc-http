package product

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/parthasarathi/go-grpc-http/gen/go/common/v1"
	productv1 "github.com/parthasarathi/go-grpc-http/gen/go/product/v1"
	"github.com/parthasarathi/go-grpc-http/internal/product/service"
)

type Server struct {
	productv1.UnimplementedProductServiceServer
	svc *service.ProductService
}

func NewServer(svc *service.ProductService) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.PriceCents <= 0 {
		return nil, status.Error(codes.InvalidArgument, "price must be positive")
	}

	product, err := s.svc.CreateProduct(ctx, req.Name, req.Description, req.PriceCents, req.Currency, req.StockQuantity, req.Sku)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create product: %v", err)
	}

	return &productv1.CreateProductResponse{
		ProductId:     product.ID,
		Name:          product.Name,
		PriceCents:    product.PriceCents,
		Currency:      product.Currency,
		StockQuantity: product.StockQuantity,
		Sku:           product.SKU,
		CreatedAt:     timestamppb.New(product.CreatedAt),
	}, nil
}

func (s *Server) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.GetProductResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	product, err := s.svc.GetProduct(ctx, req.ProductId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "get product: %v", err)
	}

	return &productv1.GetProductResponse{
		ProductId:     product.ID,
		Name:          product.Name,
		Description:   product.Description,
		PriceCents:    product.PriceCents,
		Currency:      product.Currency,
		StockQuantity: product.StockQuantity,
		Sku:           product.SKU,
		CreatedAt:     timestamppb.New(product.CreatedAt),
		UpdatedAt:     timestamppb.New(product.UpdatedAt),
	}, nil
}

func (s *Server) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	var page, pageSize int32 = 1, 20
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 {
			pageSize = req.Pagination.PageSize
		}
	}

	products, total, err := s.svc.ListProducts(ctx, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list products: %v", err)
	}

	resp := &productv1.ListProductsResponse{
		Pagination: &commonv1.PaginationResponse{
			TotalCount: total,
			Page:       page,
			PageSize:   pageSize,
		},
	}

	for _, p := range products {
		resp.Products = append(resp.Products, &productv1.GetProductResponse{
			ProductId:     p.ID,
			Name:          p.Name,
			Description:   p.Description,
			PriceCents:    p.PriceCents,
			Currency:      p.Currency,
			StockQuantity: p.StockQuantity,
			Sku:           p.SKU,
			CreatedAt:     timestamppb.New(p.CreatedAt),
			UpdatedAt:     timestamppb.New(p.UpdatedAt),
		})
	}

	return resp, nil
}

func (s *Server) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.UpdateProductResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	product, err := s.svc.UpdateProduct(ctx, req.ProductId, req.Name, req.Description, req.PriceCents, req.Currency)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "update product: %v", err)
	}

	return &productv1.UpdateProductResponse{
		ProductId:     product.ID,
		Name:          product.Name,
		Description:   product.Description,
		PriceCents:    product.PriceCents,
		Currency:      product.Currency,
		StockQuantity: product.StockQuantity,
		UpdatedAt:     timestamppb.New(product.UpdatedAt),
	}, nil
}

func (s *Server) UpdateInventory(ctx context.Context, req *productv1.UpdateInventoryRequest) (*productv1.UpdateInventoryResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	product, err := s.svc.UpdateInventory(ctx, req.ProductId, req.Quantity)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "update inventory: %v", err)
	}

	return &productv1.UpdateInventoryResponse{
		ProductId:     product.ID,
		StockQuantity: product.StockQuantity,
		UpdatedAt:     timestamppb.New(product.UpdatedAt),
	}, nil
}

func (s *Server) ReserveStock(ctx context.Context, req *productv1.ReserveStockRequest) (*productv1.ReserveStockResponse, error) {
	if req.ProductId == "" || req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id and positive quantity required")
	}

	remaining, err := s.svc.ReserveStock(ctx, req.ProductId, req.Quantity)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient") {
			return &productv1.ReserveStockResponse{Success: false, RemainingStock: 0}, nil
		}
		return nil, status.Errorf(codes.Internal, "reserve stock: %v", err)
	}

	return &productv1.ReserveStockResponse{
		Success:        true,
		RemainingStock: remaining,
	}, nil
}

func (s *Server) ReleaseStock(ctx context.Context, req *productv1.ReleaseStockRequest) (*productv1.ReleaseStockResponse, error) {
	if req.ProductId == "" || req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id and positive quantity required")
	}

	remaining, err := s.svc.ReleaseStock(ctx, req.ProductId, req.Quantity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "release stock: %v", err)
	}

	return &productv1.ReleaseStockResponse{
		Success:        true,
		RemainingStock: remaining,
	}, nil
}
