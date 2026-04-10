package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/parthasarathi/go-grpc-http/internal/product/model"
	"github.com/parthasarathi/go-grpc-http/internal/product/repository"
)

type ProductService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) CreateProduct(ctx context.Context, name, description string, priceCents int64, currency string, stockQty int32, sku string) (*model.Product, error) {
	now := time.Now().UTC()
	product := &model.Product{
		ID:            uuid.New().String(),
		Name:          name,
		Description:   description,
		PriceCents:    priceCents,
		Currency:      currency,
		StockQuantity: stockQty,
		SKU:           sku,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return product, nil
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*model.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	if product == nil {
		return nil, fmt.Errorf("product not found")
	}
	return product, nil
}

func (s *ProductService) ListProducts(ctx context.Context, page, pageSize int32) ([]*model.Product, int32, error) {
	products, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	return products, total, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, id, name, description string, priceCents int64, currency string) (*model.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	if product == nil {
		return nil, fmt.Errorf("product not found")
	}

	product.Name = name
	product.Description = description
	product.PriceCents = priceCents
	product.Currency = currency
	product.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	return product, nil
}

func (s *ProductService) UpdateInventory(ctx context.Context, id string, quantity int32) (*model.Product, error) {
	product, err := s.repo.UpdateStock(ctx, id, quantity)
	if err != nil {
		return nil, fmt.Errorf("update inventory: %w", err)
	}
	return product, nil
}

func (s *ProductService) ReserveStock(ctx context.Context, productID string, quantity int32) (int32, error) {
	remaining, err := s.repo.ReserveStock(ctx, productID, quantity)
	if err != nil {
		return 0, err
	}
	return remaining, nil
}

func (s *ProductService) ReleaseStock(ctx context.Context, productID string, quantity int32) (int32, error) {
	remaining, err := s.repo.ReleaseStock(ctx, productID, quantity)
	if err != nil {
		return 0, err
	}
	return remaining, nil
}
