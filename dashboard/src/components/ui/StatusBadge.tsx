import { getStatusBgColor } from '@/utils/api';

interface StatusBadgeProps {
  status: string;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span className={`
      inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
      border ${getStatusBgColor(status)}
    `}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
}
