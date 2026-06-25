import './LoadingIndicator.scss';

interface LoadingIndicatorProps {
  label: string;
  compact?: boolean;
}

// LoadingIndicator gives asynchronous screens and buttons one consistent progress state.
export function LoadingIndicator({ label, compact = false }: LoadingIndicatorProps) {
  return (
    <span
      className={compact ? 'loading-indicator loading-indicator--compact' : 'loading-indicator'}
      role="status"
      aria-live="polite"
    >
      <span className="loading-spinner" aria-hidden="true" />
      <span>{label}</span>
    </span>
  );
}
