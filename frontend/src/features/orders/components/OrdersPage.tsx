import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { ordersApi } from '../api';
import { productsApi } from '../../products/api';
import { useAuth } from '../../../context/AuthContext';
import { Button, Card, Alert, Badge, Input } from '../../../components/ui';
import { formatCents, formatDate } from '../../../lib/format';
import type { Order, Product } from '../../../types/api';

export function OrdersPage() {
  const { userId, isAuthenticated } = useAuth();
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);

  const fetchOrders = async () => {
    if (!userId) return;
    try {
      setLoading(true);
      const res = await ordersApi.listByUser(userId);
      setOrders(res.orders || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load orders');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isAuthenticated) fetchOrders();
    else setLoading(false);
  }, [isAuthenticated, userId]);

  if (!isAuthenticated) {
    return (
      <Card>
        <p className="text-center text-gray-500 py-8">
          <Link to="/login" className="text-indigo-600 hover:underline">Sign in</Link> to view your orders.
        </p>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">My Orders</h1>
        <Button onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : '+ Place Order'}
        </Button>
      </div>

      {error && <Alert type="error" message={error} onDismiss={() => setError('')} />}

      {showForm && (
        <PlaceOrderForm
          userId={userId!}
          onPlaced={() => { setShowForm(false); setTimeout(fetchOrders, 2000); }}
          onError={setError}
        />
      )}

      {loading ? (
        <p className="text-gray-500">Loading orders...</p>
      ) : orders.length === 0 ? (
        <Card><p className="text-gray-500 text-center py-8">No orders yet.</p></Card>
      ) : (
        <div className="space-y-4">
          {orders.map(order => (
            <Card key={order.orderId}>
              <div className="flex items-center justify-between mb-3">
                <div>
                  <Link to={`/orders/${order.orderId}`} className="font-mono text-sm text-indigo-600 hover:underline">
                    {order.orderId.slice(0, 8)}...
                  </Link>
                  <span className="text-xs text-gray-400 ml-3">{formatDate(order.createdAt)}</span>
                </div>
                <div className="flex items-center space-x-3">
                  <span className="font-semibold">{formatCents(order.totalCents, order.currency)}</span>
                  <Badge status={order.status} />
                </div>
              </div>
              <div className="text-sm text-gray-500">
                {(order.items || []).map((item, i) => (
                  <span key={i}>
                    {item.productName || item.productId.slice(0, 8)} x{item.quantity}
                    {i < order.items.length - 1 ? ', ' : ''}
                  </span>
                ))}
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function PlaceOrderForm({ userId, onPlaced, onError }: { userId: string; onPlaced: () => void; onError: (msg: string) => void }) {
  const [products, setProducts] = useState<Product[]>([]);
  const [selectedProduct, setSelectedProduct] = useState('');
  const [quantity, setQuantity] = useState('1');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    productsApi.list().then(res => setProducts(res.products || []));
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProduct) { onError('Select a product'); return; }
    setLoading(true);
    try {
      await ordersApi.place({
        user_id: userId,
        items: [{ product_id: selectedProduct, quantity: parseInt(quantity, 10) }],
        currency: 'USD',
      });
      onPlaced();
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Failed to place order');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card title="Place Order">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Product</label>
          <select
            value={selectedProduct}
            onChange={e => setSelectedProduct(e.target.value)}
            className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
            required
          >
            <option value="">Select a product...</option>
            {products.map(p => (
              <option key={p.productId} value={p.productId}>
                {p.name} - {formatCents(p.priceCents)} ({p.stockQuantity} in stock)
              </option>
            ))}
          </select>
        </div>
        <Input label="Quantity" type="number" min="1" value={quantity} onChange={e => setQuantity(e.target.value)} required />
        <Button type="submit" loading={loading}>Place Order</Button>
      </form>
    </Card>
  );
}
