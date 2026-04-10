import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../../../context/AuthContext';
import { authApi } from '../../auth/api';
import { productsApi } from '../../products/api';
import { ordersApi } from '../../orders/api';
import { Card } from '../../../components/ui';
import { formatCents, formatDate } from '../../../lib/format';
import type { User, Order } from '../../../types/api';

export function DashboardPage() {
  const { userId, isAuthenticated } = useAuth();
  const [user, setUser] = useState<User | null>(null);
  const [productCount, setProductCount] = useState(0);
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!userId || !isAuthenticated) {
      setLoading(false);
      return;
    }

    const fetchData = async () => {
      try {
        const [u, p, o] = await Promise.all([
          authApi.getUser(userId),
          productsApi.list(),
          ordersApi.listByUser(userId),
        ]);
        setUser(u);
        setProductCount(p.pagination?.totalCount || 0);
        setOrders(o.orders || []);
      } catch {
        // Silently handle - partial data is ok for dashboard
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [userId]);

  if (loading) return <p className="text-gray-500">Loading dashboard...</p>;

  if (!isAuthenticated) {
    return (
      <div className="text-center py-16">
        <h1 className="text-3xl font-bold text-gray-900 mb-4">Welcome to OrderFlow</h1>
        <p className="text-gray-500 mb-6">A microservice demo with Go, gRPC, and React.</p>
        <div className="flex justify-center space-x-4">
          <Link to="/login" className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700">Sign In</Link>
          <Link to="/register" className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50">Register</Link>
        </div>
        <p className="text-sm text-gray-400 mt-8">Or <Link to="/products" className="text-indigo-600 hover:underline">browse products</Link> without signing in.</p>
      </div>
    );
  }

  const totalSpent = orders.reduce((sum, o) => sum + parseInt(o.totalCents || '0', 10), 0);
  const paidOrders = orders.filter(o => o.status === 'paid').length;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">
        Welcome back, {user?.firstName || 'User'}
      </h1>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatCard label="Products" value={productCount.toString()} />
        <StatCard label="Total Orders" value={orders.length.toString()} />
        <StatCard label="Completed" value={paidOrders.toString()} />
        <StatCard label="Total Spent" value={formatCents(totalSpent)} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card title="Recent Orders">
          {orders.length === 0 ? (
            <p className="text-sm text-gray-500 py-4">No orders yet. <Link to="/orders" className="text-indigo-600 hover:underline">Place one</Link>.</p>
          ) : (
            <div className="space-y-3">
              {orders.slice(0, 5).map(o => (
                <div key={o.orderId} className="flex items-center justify-between text-sm">
                  <Link to={`/orders/${o.orderId}`} className="text-indigo-600 hover:underline font-mono">
                    {o.orderId.slice(0, 8)}...
                  </Link>
                  <div className="flex items-center space-x-3">
                    <span>{formatCents(o.totalCents)}</span>
                    <span className={`text-xs px-2 py-0.5 rounded-full capitalize ${
                      o.status === 'paid' ? 'bg-green-100 text-green-700' :
                      o.status === 'pending' ? 'bg-yellow-100 text-yellow-700' :
                      'bg-gray-100 text-gray-700'
                    }`}>{o.status}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>

        <Card title="Account">
          <dl className="space-y-3 text-sm">
            <div className="flex justify-between">
              <dt className="text-gray-500">Email</dt>
              <dd className="font-medium">{user?.email}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Name</dt>
              <dd className="font-medium">{user?.firstName} {user?.lastName}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Member Since</dt>
              <dd className="font-medium">{formatDate(user?.createdAt || '')}</dd>
            </div>
          </dl>
        </Card>
      </div>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6 text-center">
      <p className="text-sm text-gray-500">{label}</p>
      <p className="text-2xl font-bold text-gray-900 mt-1">{value}</p>
    </div>
  );
}
