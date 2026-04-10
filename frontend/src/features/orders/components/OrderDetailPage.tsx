import { useEffect, useState, useRef } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ordersApi } from '../api';
import { paymentsApi } from '../../payments/api';
import { Card, Badge, Alert, Button } from '../../../components/ui';
import { formatCents, formatDate } from '../../../lib/format';
import type { Order, Payment } from '../../../types/api';

export function OrderDetailPage() {
  const { orderId } = useParams<{ orderId: string }>();
  const [order, setOrder] = useState<Order | null>(null);
  const [payments, setPayments] = useState<Payment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refunding, setRefunding] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchData = async () => {
    if (!orderId) return;
    try {
      const [o, p] = await Promise.all([
        ordersApi.get(orderId),
        paymentsApi.listByOrder(orderId),
      ]);
      setOrder(o);
      setPayments(p.payments || []);
      return o;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load order');
      return null;
    } finally {
      setLoading(false);
    }
  };

  // Initial fetch + auto-poll when status is pending
  useEffect(() => {
    fetchData().then(o => {
      if (o && o.status === 'pending') {
        pollRef.current = setInterval(async () => {
          const updated = await fetchData();
          if (updated && updated.status !== 'pending') {
            if (pollRef.current) clearInterval(pollRef.current);
          }
        }, 2000);
      }
    });

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [orderId]);

  const handleCancel = async () => {
    if (!orderId) return;
    try {
      await ordersApi.cancel(orderId, 'Cancelled by user');
      fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel order');
    }
  };

  const handleRefund = async (paymentId: string) => {
    setRefunding(paymentId);
    try {
      await paymentsApi.refund(paymentId, 'Refund requested by user');
      fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Refund failed');
    } finally {
      setRefunding(null);
    }
  };

  if (loading) {
    return (
      <div className="animate-pulse space-y-6">
        <div className="h-4 bg-gray-200 rounded w-32" />
        <div className="h-8 bg-gray-200 rounded w-64" />
        <div className="grid grid-cols-3 gap-6">
          <div className="col-span-2 h-48 bg-gray-200 rounded-xl" />
          <div className="h-48 bg-gray-200 rounded-xl" />
        </div>
      </div>
    );
  }
  if (error && !order) return <Alert type="error" message={error} />;
  if (!order) return <Alert type="error" message="Order not found" />;

  return (
    <div className="space-y-6">
      {error && <Alert type="error" message={error} onDismiss={() => setError('')} />}

      <div className="flex items-center justify-between">
        <div>
          <Link to="/orders" className="inline-flex items-center text-sm text-indigo-600 hover:text-indigo-800">
            <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" /></svg>
            Back to Orders
          </Link>
          <h1 className="text-2xl font-bold text-gray-900 mt-1">Order #{order.orderId.slice(0, 8)}</h1>
        </div>
        <div className="flex items-center space-x-3">
          <Button variant="secondary" size="sm" onClick={() => fetchData()}>Refresh</Button>
          {order.status === 'pending' && (
            <Button variant="danger" size="sm" onClick={handleCancel}>Cancel Order</Button>
          )}
        </div>
      </div>

      {/* Status timeline */}
      {order.status === 'pending' && (
        <div className="flex items-center space-x-3 p-4 bg-amber-50 rounded-xl border border-amber-100">
          <svg className="w-5 h-5 text-amber-500 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          <span className="text-sm font-medium text-amber-800">Processing payment... This page updates automatically.</span>
        </div>
      )}
      {order.status === 'paid' && (
        <div className="flex items-center space-x-3 p-4 bg-green-50 rounded-xl border border-green-100">
          <svg className="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
          <span className="text-sm font-medium text-green-800">Payment confirmed! Your order is complete.</span>
        </div>
      )}
      {order.status === 'cancelled' && (
        <div className="flex items-center space-x-3 p-4 bg-red-50 rounded-xl border border-red-100">
          <svg className="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
          <span className="text-sm font-medium text-red-800">This order has been cancelled.</span>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card title="Details" className="lg:col-span-2">
          <dl className="grid grid-cols-2 gap-6 text-sm">
            <div>
              <dt className="text-gray-500">Status</dt>
              <dd className="mt-1"><Badge status={order.status} /></dd>
            </div>
            <div>
              <dt className="text-gray-500">Total</dt>
              <dd className="mt-1 text-xl font-bold text-gray-900">{formatCents(order.totalCents, order.currency)}</dd>
            </div>
            <div>
              <dt className="text-gray-500">Order ID</dt>
              <dd className="mt-1 font-mono text-xs text-gray-700">{order.orderId}</dd>
            </div>
            <div>
              <dt className="text-gray-500">Payment ID</dt>
              <dd className="mt-1 font-mono text-xs text-gray-700">{order.paymentId || '-'}</dd>
            </div>
            <div>
              <dt className="text-gray-500">Created</dt>
              <dd className="mt-1">{formatDate(order.createdAt)}</dd>
            </div>
            <div>
              <dt className="text-gray-500">Updated</dt>
              <dd className="mt-1">{formatDate(order.updatedAt)}</dd>
            </div>
          </dl>
        </Card>

        <Card title="Items">
          <div className="space-y-4">
            {(order.items || []).map((item, i) => (
              <div key={i} className="flex justify-between items-center">
                <div>
                  <Link to={`/products/${item.productId}`} className="text-sm font-medium text-indigo-600 hover:underline">
                    {item.productName || item.productId.slice(0, 8)}
                  </Link>
                  <p className="text-xs text-gray-400">Qty: {item.quantity}</p>
                </div>
                <span className="font-semibold text-sm">{formatCents(String(parseInt(item.priceCents, 10) * item.quantity))}</span>
              </div>
            ))}
            <div className="pt-3 border-t border-gray-100 flex justify-between">
              <span className="font-semibold text-gray-900">Total</span>
              <span className="font-bold text-lg text-gray-900">{formatCents(order.totalCents, order.currency)}</span>
            </div>
          </div>
        </Card>
      </div>

      {/* Payments table */}
      {payments.length > 0 && (
        <Card title="Payments">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-gray-500 border-b">
                  <th className="pb-3 font-medium">Payment ID</th>
                  <th className="pb-3 font-medium">Amount</th>
                  <th className="pb-3 font-medium">Method</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium">Date</th>
                  <th className="pb-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {payments.map(p => (
                  <tr key={p.paymentId} className="border-b last:border-0">
                    <td className="py-3 font-mono text-xs">{p.paymentId.slice(0, 12)}...</td>
                    <td className="py-3 font-semibold">{formatCents(p.amountCents, p.currency)}</td>
                    <td className="py-3 capitalize">{p.method}</td>
                    <td className="py-3"><Badge status={p.status} /></td>
                    <td className="py-3 text-gray-500">{formatDate(p.createdAt)}</td>
                    <td className="py-3">
                      {p.status === 'completed' && (
                        <Button
                          variant="danger"
                          size="sm"
                          loading={refunding === p.paymentId}
                          onClick={() => handleRefund(p.paymentId)}
                        >
                          Refund
                        </Button>
                      )}
                      {p.status === 'refunded' && (
                        <span className="text-xs text-gray-400">Refunded</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  );
}
