import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { authApi } from '../api';
import { Button, Input, Card, Alert } from '../../../components/ui';

export function RegisterPage() {
  const [form, setForm] = useState({ email: '', password: '', first_name: '', last_name: '' });
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const update = (field: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm(prev => ({ ...prev, [field]: e.target.value }));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await authApi.register(form);
      setSuccess(`Account created! Your auth token: ${res.authToken}`);
      setTimeout(() => navigate('/login'), 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-[70vh]">
      <Card title="Create Account" className="w-full max-w-md">
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <Alert type="error" message={error} onDismiss={() => setError('')} />}
          {success && <Alert type="success" message={success} />}
          <div className="grid grid-cols-2 gap-4">
            <Input label="First Name" value={form.first_name} onChange={update('first_name')} required />
            <Input label="Last Name" value={form.last_name} onChange={update('last_name')} required />
          </div>
          <Input label="Email" type="email" value={form.email} onChange={update('email')} required />
          <Input label="Password" type="password" value={form.password} onChange={update('password')} required minLength={6} />
          <Button type="submit" loading={loading} className="w-full">Create Account</Button>
          <p className="text-center text-sm text-gray-500">
            Already have an account?{' '}
            <Link to="/login" className="text-indigo-600 hover:text-indigo-800 font-medium">Sign In</Link>
          </p>
        </form>
      </Card>
    </div>
  );
}
