import { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { productsApi } from '../api';
import { ordersApi } from '../../orders/api';
import { useAuth } from '../../../context/AuthContext';
import { Button, Alert, Input } from '../../../components/ui';
import { formatCents } from '../../../lib/format';
import { getProductImages } from '../../../lib/product-images';
import type { Product } from '../../../types/api';

export function ProductDetailPage() {
  const { productId } = useParams<{ productId: string }>();
  const { isAuthenticated, userId } = useAuth();
  const navigate = useNavigate();
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeImg, setActiveImg] = useState(0);

  // Order flow
  const [orderStep, setOrderStep] = useState<'closed' | 'quantity' | 'payment'>('closed');
  const [quantity, setQuantity] = useState('1');
  const [ordering, setOrdering] = useState(false);
  const [orderSuccess, setOrderSuccess] = useState('');
  const [card, setCard] = useState({ number: '4242 4242 4242 4242', expiry: '12/28', cvv: '123', name: '' });

  // Edit flow
  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState({ name: '', description: '', price: '', currency: '' });
  const [saving, setSaving] = useState(false);

  // Inventory flow
  const [showInventory, setShowInventory] = useState(false);
  const [newStock, setNewStock] = useState('');
  const [updatingStock, setUpdatingStock] = useState(false);

  const fetchProduct = async () => {
    if (!productId) return;
    try {
      const p = await productsApi.get(productId);
      setProduct(p);
      setEditForm({
        name: p.name,
        description: p.description,
        price: (parseInt(p.priceCents, 10) / 100).toFixed(2),
        currency: p.currency,
      });
      setNewStock(String(p.stockQuantity));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load product');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchProduct(); }, [productId]);

  const handlePayAndOrder = async () => {
    if (!userId || !productId) return;
    if (!card.name.trim()) { setError('Cardholder name is required'); return; }
    setOrdering(true);
    setError('');
    try {
      const res = await ordersApi.place({
        user_id: userId,
        items: [{ product_id: productId, quantity: parseInt(quantity, 10) }],
        currency: 'USD',
      });
      setOrderSuccess(res.orderId);
      setOrderStep('closed');
      fetchProduct();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Payment failed');
    } finally {
      setOrdering(false);
    }
  };

  const handleUpdate = async () => {
    if (!productId) return;
    setSaving(true);
    try {
      await productsApi.update(productId, {
        name: editForm.name,
        description: editForm.description,
        price_cents: Math.round(parseFloat(editForm.price) * 100),
        currency: editForm.currency,
      });
      setEditing(false);
      fetchProduct();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Update failed');
    } finally {
      setSaving(false);
    }
  };

  const handleInventory = async () => {
    if (!productId) return;
    setUpdatingStock(true);
    try {
      await productsApi.updateInventory(productId, parseInt(newStock, 10));
      setShowInventory(false);
      fetchProduct();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Inventory update failed');
    } finally {
      setUpdatingStock(false);
    }
  };

  if (loading) {
    return (
      <div className="animate-pulse space-y-6">
        <div className="h-4 bg-gray-200 rounded w-24" />
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-10">
          <div className="h-96 bg-gray-200 rounded-2xl" />
          <div className="space-y-4">
            <div className="h-8 bg-gray-200 rounded w-3/4" />
            <div className="h-4 bg-gray-200 rounded w-1/2" />
            <div className="h-20 bg-gray-200 rounded" />
            <div className="h-10 bg-gray-200 rounded w-1/3" />
          </div>
        </div>
      </div>
    );
  }

  if (!product) return <Alert type="error" message="Product not found" />;

  const images = getProductImages(product.sku);
  const price = parseInt(product.priceCents, 10);
  const rating = ((price % 15) / 10 + 3.5).toFixed(1);
  const reviews = (price % 200) + 12;

  return (
    <div className="space-y-6">
      <Link to="/products" className="inline-flex items-center text-sm text-indigo-600 hover:text-indigo-800">
        <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" /></svg>
        Back to Shop
      </Link>

      {error && <Alert type="error" message={error} onDismiss={() => setError('')} />}

      {orderSuccess && (
        <Alert type="success" message={`Order placed! ID: ${orderSuccess.slice(0, 8)}... Your order is being processed.`} onDismiss={() => setOrderSuccess('')} />
      )}
      {orderSuccess && (
        <div className="flex space-x-3">
          <Button onClick={() => navigate(`/orders/${orderSuccess}`)}>View Order Status</Button>
          <Button variant="secondary" onClick={() => navigate('/orders')}>All Orders</Button>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-10">
        {/* Image Gallery */}
        <div>
          <div className="rounded-2xl overflow-hidden bg-gray-100 mb-4">
            <img src={images.gallery[activeImg]} alt={product.name} className="w-full h-96 object-cover" />
          </div>
          <div className="flex space-x-3">
            {images.gallery.map((img, i) => (
              <button
                key={i}
                onClick={() => setActiveImg(i)}
                className={`rounded-xl overflow-hidden border-2 transition-all ${
                  i === activeImg ? 'border-indigo-500 shadow-md ring-2 ring-indigo-200' : 'border-gray-200 opacity-60 hover:opacity-100'
                }`}
              >
                <img src={img} alt="" className="w-20 h-20 object-cover" />
              </button>
            ))}
          </div>
        </div>

        {/* Product Info */}
        <div className="space-y-6">
          <div>
            <span className="text-xs font-medium text-indigo-600 uppercase tracking-wider">{product.sku}</span>
            {!editing ? (
              <h1 className="text-3xl font-bold text-gray-900 mt-1">{product.name}</h1>
            ) : (
              <Input label="Name" value={editForm.name} onChange={e => setEditForm(prev => ({ ...prev, name: e.target.value }))} className="mt-2" />
            )}
          </div>

          {/* Rating */}
          <div className="flex items-center space-x-2">
            <div className="flex">
              {[1, 2, 3, 4, 5].map(star => (
                <svg key={star} className={`w-5 h-5 ${star <= Math.round(parseFloat(rating)) ? 'text-amber-400' : 'text-gray-200'}`} fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                </svg>
              ))}
            </div>
            <span className="text-sm text-gray-500">{rating} ({reviews} reviews)</span>
          </div>

          {/* Description */}
          {!editing ? (
            <p className="text-gray-600 leading-relaxed">{product.description}</p>
          ) : (
            <Input label="Description" value={editForm.description} onChange={e => setEditForm(prev => ({ ...prev, description: e.target.value }))} />
          )}

          {/* Price */}
          {!editing ? (
            <div className="text-4xl font-bold text-gray-900">{formatCents(product.priceCents, product.currency)}</div>
          ) : (
            <div className="grid grid-cols-2 gap-4">
              <Input label="Price ($)" type="number" step="0.01" value={editForm.price} onChange={e => setEditForm(prev => ({ ...prev, price: e.target.value }))} />
              <Input label="Currency" value={editForm.currency} onChange={e => setEditForm(prev => ({ ...prev, currency: e.target.value }))} />
            </div>
          )}

          {/* Stock */}
          <div className="flex items-center space-x-3 p-4 rounded-xl bg-gray-50">
            <span className={`w-3 h-3 rounded-full ${
              product.stockQuantity > 20 ? 'bg-green-500' : product.stockQuantity > 0 ? 'bg-amber-500 animate-pulse' : 'bg-red-500'
            }`} />
            <span className="text-sm font-medium text-gray-700">
              {product.stockQuantity > 20 ? `In Stock - ${product.stockQuantity} available`
                : product.stockQuantity > 0 ? `Low Stock - Only ${product.stockQuantity} left!`
                : 'Out of Stock'}
            </span>
          </div>

          {/* Inventory Update */}
          {isAuthenticated && showInventory && (
            <div className="p-4 rounded-xl border border-gray-200 space-y-3">
              <Input label="New Stock Quantity" type="number" min="0" value={newStock} onChange={e => setNewStock(e.target.value)} />
              <div className="flex space-x-2">
                <Button onClick={handleInventory} loading={updatingStock} size="sm">Update Stock</Button>
                <Button variant="secondary" size="sm" onClick={() => setShowInventory(false)}>Cancel</Button>
              </div>
            </div>
          )}

          {/* Edit save/cancel */}
          {editing && (
            <div className="flex space-x-3">
              <Button onClick={handleUpdate} loading={saving}>Save Changes</Button>
              <Button variant="secondary" onClick={() => setEditing(false)}>Cancel</Button>
            </div>
          )}

          {/* Order Now */}
          {!editing && orderStep === 'closed' && !orderSuccess && (
            <div className="space-y-3 pt-2">
              {isAuthenticated ? (
                <Button
                  size="lg"
                  className="w-full"
                  onClick={() => setOrderStep('quantity')}
                  disabled={product.stockQuantity === 0}
                >
                  {product.stockQuantity > 0 ? 'Order Now' : 'Out of Stock'}
                </Button>
              ) : (
                <Link to="/login" className="block w-full text-center bg-indigo-600 text-white py-3 rounded-lg font-medium hover:bg-indigo-700 transition-colors">
                  Sign in to Order
                </Link>
              )}

              {isAuthenticated && (
                <div className="flex space-x-2">
                  <Button variant="secondary" size="sm" onClick={() => setEditing(true)}>Edit Product</Button>
                  <Button variant="secondary" size="sm" onClick={() => setShowInventory(!showInventory)}>Update Stock</Button>
                </div>
              )}
            </div>
          )}

          {/* Step 1: Quantity selection */}
          {orderStep === 'quantity' && (
            <div className="p-6 rounded-2xl border-2 border-indigo-100 bg-indigo-50/30 space-y-4">
              <div className="flex items-center space-x-2">
                <span className="w-7 h-7 rounded-full bg-indigo-600 text-white text-sm font-bold flex items-center justify-center">1</span>
                <h3 className="font-semibold text-gray-900">Select Quantity</h3>
              </div>
              <Input
                label="Quantity"
                type="number"
                min="1"
                max={String(product.stockQuantity)}
                value={quantity}
                onChange={e => setQuantity(e.target.value)}
              />
              <div className="text-sm text-gray-600">
                Subtotal: <span className="font-bold text-gray-900 text-lg">
                  {formatCents(String(parseInt(product.priceCents, 10) * parseInt(quantity || '0', 10)))}
                </span>
              </div>
              <div className="flex space-x-3">
                <Button onClick={() => setOrderStep('payment')} size="lg" className="flex-1">
                  Proceed to Payment
                </Button>
                <Button variant="secondary" size="lg" onClick={() => setOrderStep('closed')}>Cancel</Button>
              </div>
            </div>
          )}

          {/* Step 2: Payment form */}
          {orderStep === 'payment' && (
            <div className="p-6 rounded-2xl border-2 border-green-100 bg-green-50/30 space-y-5">
              <div className="flex items-center space-x-2">
                <span className="w-7 h-7 rounded-full bg-green-600 text-white text-sm font-bold flex items-center justify-center">2</span>
                <h3 className="font-semibold text-gray-900">Payment Details</h3>
              </div>

              {/* Card visual */}
              <div className="bg-gradient-to-br from-gray-800 to-gray-900 rounded-xl p-5 text-white shadow-lg">
                <div className="flex justify-between items-start mb-8">
                  <span className="text-xs uppercase tracking-widest opacity-70">Credit Card</span>
                  <svg className="w-10 h-8" viewBox="0 0 48 32" fill="none">
                    <circle cx="16" cy="16" r="14" fill="#EB001B" opacity="0.8" />
                    <circle cx="32" cy="16" r="14" fill="#F79E1B" opacity="0.8" />
                  </svg>
                </div>
                <div className="font-mono text-lg tracking-widest mb-4">{card.number}</div>
                <div className="flex justify-between text-sm">
                  <div>
                    <span className="text-xs uppercase opacity-50 block">Cardholder</span>
                    <span>{card.name || 'YOUR NAME'}</span>
                  </div>
                  <div>
                    <span className="text-xs uppercase opacity-50 block">Expires</span>
                    <span>{card.expiry}</span>
                  </div>
                </div>
              </div>

              {/* Card form */}
              <Input
                label="Cardholder Name"
                value={card.name}
                onChange={e => setCard(prev => ({ ...prev, name: e.target.value }))}
                placeholder="John Doe"
                required
              />
              <Input
                label="Card Number"
                value={card.number}
                onChange={e => setCard(prev => ({ ...prev, number: e.target.value }))}
                placeholder="4242 4242 4242 4242"
              />
              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="Expiry"
                  value={card.expiry}
                  onChange={e => setCard(prev => ({ ...prev, expiry: e.target.value }))}
                  placeholder="MM/YY"
                />
                <Input
                  label="CVV"
                  type="password"
                  value={card.cvv}
                  onChange={e => setCard(prev => ({ ...prev, cvv: e.target.value }))}
                  placeholder="123"
                  maxLength={4}
                />
              </div>

              <div className="bg-white rounded-lg p-4 border border-gray-200">
                <div className="flex justify-between text-sm text-gray-600">
                  <span>{product.name} x {quantity}</span>
                  <span>{formatCents(String(parseInt(product.priceCents, 10) * parseInt(quantity || '0', 10)))}</span>
                </div>
                <div className="flex justify-between text-sm text-gray-600 mt-1">
                  <span>Processing fee</span>
                  <span>$0.00</span>
                </div>
                <div className="border-t border-gray-100 mt-3 pt-3 flex justify-between font-bold text-gray-900">
                  <span>Total</span>
                  <span className="text-lg">{formatCents(String(parseInt(product.priceCents, 10) * parseInt(quantity || '0', 10)))}</span>
                </div>
              </div>

              <p className="text-xs text-gray-400 flex items-center space-x-1">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" /></svg>
                <span>Secure payment. This is a demo - no real charges will be made.</span>
              </p>

              <div className="flex space-x-3">
                <Button onClick={handlePayAndOrder} loading={ordering} size="lg" className="flex-1">
                  {ordering ? 'Processing...' : `Pay ${formatCents(String(parseInt(product.priceCents, 10) * parseInt(quantity || '0', 10)))}`}
                </Button>
                <Button variant="secondary" size="lg" onClick={() => setOrderStep('quantity')}>Back</Button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
