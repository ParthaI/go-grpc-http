import { statusColor } from '../../lib/format';

interface BadgeProps {
  status: string;
}

export function Badge({ status }: BadgeProps) {
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${statusColor(status)}`}>
      {status}
    </span>
  );
}
