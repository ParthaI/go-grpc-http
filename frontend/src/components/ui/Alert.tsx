interface AlertProps {
  type: 'success' | 'error' | 'info';
  message: string;
  onDismiss?: () => void;
}

const styles = {
  success: 'bg-green-50 text-green-800 border-green-200',
  error: 'bg-red-50 text-red-800 border-red-200',
  info: 'bg-blue-50 text-blue-800 border-blue-200',
};

export function Alert({ type, message, onDismiss }: AlertProps) {
  return (
    <div className={`rounded-lg border px-4 py-3 text-sm flex items-center justify-between ${styles[type]}`}>
      <span>{message}</span>
      {onDismiss && (
        <button onClick={onDismiss} className="ml-4 text-lg leading-none hover:opacity-70">&times;</button>
      )}
    </div>
  );
}
