import { Link, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export function AppLayout() {
  const { isAuthenticated, email, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center space-x-8">
              <Link to="/" className="text-xl font-bold text-indigo-600">
                OrderFlow
              </Link>
              {isAuthenticated && (
                <>
                  <Link to="/dashboard" className="text-sm font-medium text-gray-600 hover:text-gray-900">
                    Dashboard
                  </Link>
                  <Link to="/products" className="text-sm font-medium text-gray-600 hover:text-gray-900">
                    Products
                  </Link>
                  <Link to="/orders" className="text-sm font-medium text-gray-600 hover:text-gray-900">
                    Orders
                  </Link>
                </>
              )}
            </div>
            <div className="flex items-center space-x-4">
              {isAuthenticated ? (
                <>
                  <span className="text-sm text-gray-500">{email}</span>
                  <button
                    onClick={handleLogout}
                    className="text-sm font-medium text-red-600 hover:text-red-800"
                  >
                    Logout
                  </button>
                </>
              ) : (
                <>
                  <Link to="/login" className="text-sm font-medium text-gray-600 hover:text-gray-900">
                    Login
                  </Link>
                  <Link to="/register" className="text-sm font-medium text-indigo-600 hover:text-indigo-800">
                    Register
                  </Link>
                </>
              )}
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>
    </div>
  );
}
