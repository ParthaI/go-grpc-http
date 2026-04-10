import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { productsApi } from '../api';
import { useAuth } from '../../../context/AuthContext';
import { Button, Alert, Input } from '../../../components/ui';
import { formatCents } from '../../../lib/format';
import { getProductImages } from '../../../lib/product-images';
import type { Product } from '../../../types/api';

export function ProductsPage() {
  const { isAuthenticated } = useAuth();
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const navigate = useNavigate();

  const fetchProducts = async () => {
    try {
      setLoading(true);
      const res = await productsApi.list();
      setProducts(res.products || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load products');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchProducts(); }, []);

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Shop</h1>
          <p className="text-gray-500 mt-1">{products.length} products available</p>
        </div>
        {isAuthenticated && (
          <Button onClick={() => setShowForm(!showForm)} size="lg">
            {showForm ? 'Cancel' : '+ Add Product'}
          </Button>
        )}
      </div>

      {error && <Alert type="error" message={error} onDismiss={() => setError('')} />}

      {showForm && (
        <CreateProductForm
          onCreated={() => { setShowForm(false); fetchProducts(); }}
          onError={setError}
        />
      )}

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="bg-white rounded-2xl overflow-hidden animate-pulse">
              <div className="h-56 bg-gray-200" />
              <div className="p-5 space-y-3">
                <div className="h-4 bg-gray-200 rounded w-3/4" />
                <div className="h-3 bg-gray-200 rounded w-1/2" />
                <div className="h-6 bg-gray-200 rounded w-1/3" />
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {products.map(p => (
            <ProductCard
              key={p.productId}
              product={p}
              onClick={() => navigate(`/products/${p.productId}`)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function ProductCard({ product, onClick }: { product: Product; onClick: () => void }) {
  const [imgIndex, setImgIndex] = useState(0);
  const images = getProductImages(product.sku);
  const price = parseInt(product.priceCents, 10);
  const rating = ((price % 15) / 10 + 3.5).toFixed(1);
  const reviews = (price % 200) + 12;

  return (
    <div
      onClick={onClick}
      className="group bg-white rounded-2xl overflow-hidden shadow-sm border border-gray-100 hover:shadow-xl hover:border-gray-200 transition-all duration-300 cursor-pointer hover:-translate-y-1"
    >
      <div className="relative h-56 bg-gray-100 overflow-hidden">
        <img
          src={images.gallery[imgIndex]}
          alt={product.name}
          className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
        />
        <div className="absolute bottom-3 left-1/2 -translate-x-1/2 flex space-x-1.5">
          {images.gallery.slice(0, 5).map((_, i) => (
            <button
              key={i}
              onClick={e => { e.stopPropagation(); setImgIndex(i); }}
              className={`w-2 h-2 rounded-full transition-all ${
                i === imgIndex ? 'bg-white w-4' : 'bg-white/50 hover:bg-white/80'
              }`}
            />
          ))}
        </div>
        {product.stockQuantity <= 10 && product.stockQuantity > 0 && (
          <span className="absolute top-3 left-3 bg-amber-500 text-white text-xs font-semibold px-2.5 py-1 rounded-full">
            Only {product.stockQuantity} left
          </span>
        )}
        {product.stockQuantity === 0 && (
          <span className="absolute top-3 left-3 bg-red-500 text-white text-xs font-semibold px-2.5 py-1 rounded-full">
            Sold Out
          </span>
        )}
        <div className="absolute top-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity">
          <span className="bg-white/90 backdrop-blur-sm text-gray-700 text-xs font-medium px-3 py-1.5 rounded-full shadow-sm">
            View details
          </span>
        </div>
      </div>

      <div className="p-5">
        <h3 className="font-semibold text-gray-900 group-hover:text-indigo-600 transition-colors line-clamp-1">
          {product.name}
        </h3>
        <p className="text-sm text-gray-500 mt-1 line-clamp-2">{product.description}</p>

        <div className="flex items-center mt-3 space-x-1">
          <div className="flex">
            {[1, 2, 3, 4, 5].map(star => (
              <svg key={star} className={`w-4 h-4 ${star <= Math.round(parseFloat(rating)) ? 'text-amber-400' : 'text-gray-200'}`} fill="currentColor" viewBox="0 0 20 20">
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
              </svg>
            ))}
          </div>
          <span className="text-xs text-gray-500">{rating} ({reviews})</span>
        </div>

        <div className="flex items-end justify-between mt-4">
          <span className="text-2xl font-bold text-gray-900">
            {formatCents(product.priceCents, product.currency)}
          </span>
          <span className={`text-xs font-medium px-2 py-1 rounded-full ${
            product.stockQuantity > 20 ? 'bg-green-50 text-green-700'
              : product.stockQuantity > 0 ? 'bg-amber-50 text-amber-700'
              : 'bg-red-50 text-red-700'
          }`}>
            {product.stockQuantity > 0 ? `${product.stockQuantity} in stock` : 'Out of stock'}
          </span>
        </div>
      </div>
    </div>
  );
}

function CreateProductForm({ onCreated, onError }: { onCreated: () => void; onError: (msg: string) => void }) {
  const [form, setForm] = useState({ name: '', description: '', price: '', currency: 'USD', stock: '', sku: '' });
  const [loading, setLoading] = useState(false);

  const update = (field: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm(prev => ({ ...prev, [field]: e.target.value }));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await productsApi.create({
        name: form.name,
        description: form.description,
        price_cents: Math.round(parseFloat(form.price) * 100),
        currency: form.currency,
        stock_quantity: parseInt(form.stock, 10),
        sku: form.sku,
      });
      onCreated();
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Failed to create product');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-2xl border border-gray-200 shadow-sm p-8">
      <h3 className="text-lg font-semibold text-gray-900 mb-6">Add New Product</h3>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <Input label="Product Name" value={form.name} onChange={update('name')} required />
          <Input label="SKU" value={form.sku} onChange={update('sku')} required placeholder="e.g. PROD-001" />
        </div>
        <Input label="Description" value={form.description} onChange={update('description')} />
        <div className="grid grid-cols-3 gap-4">
          <Input label="Price ($)" type="number" step="0.01" min="0.01" value={form.price} onChange={update('price')} required />
          <Input label="Currency" value={form.currency} onChange={update('currency')} required />
          <Input label="Stock Quantity" type="number" min="0" value={form.stock} onChange={update('stock')} required />
        </div>
        <div className="pt-2"><Button type="submit" loading={loading} size="lg">Create Product</Button></div>
      </form>
    </div>
  );
}
