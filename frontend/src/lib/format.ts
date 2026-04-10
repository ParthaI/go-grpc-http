/**
 * Formatting utilities used across the application.
 */

export function formatCents(cents: string | number, currency = 'USD'): string {
  const amount = typeof cents === 'string' ? parseInt(cents, 10) : cents;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
  }).format(amount / 100);
}

export function formatDate(iso: string): string {
  if (!iso) return '-';
  return new Date(iso).toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function statusColor(status: string): string {
  switch (status) {
    case 'pending': return 'bg-yellow-100 text-yellow-800';
    case 'paid': case 'completed': return 'bg-green-100 text-green-800';
    case 'cancelled': case 'failed': return 'bg-red-100 text-red-800';
    case 'refunded': return 'bg-gray-100 text-gray-800';
    default: return 'bg-gray-100 text-gray-800';
  }
}
