import { Component, type ReactNode } from "react";
import { AlertTriangle, RotateCcw } from "lucide-react";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

// ErrorBoundary catches rendering errors and shows a fallback UI.
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("[ErrorBoundary] Caught:", error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="flex flex-col items-center justify-center h-full p-8 text-center">
          <div className="w-16 h-16 rounded-2xl bg-red-500/10 flex items-center justify-center mb-4">
            <AlertTriangle size={32} className="text-red-400" />
          </div>
          <h2 className="text-xl font-bold text-gray-200 mb-2">
            Something went wrong
          </h2>
          <p className="text-sm text-gray-500 max-w-md mb-4">
            {this.state.error?.message || "An unexpected error occurred"}
          </p>
          <button
            onClick={this.handleReset}
            className="btn-primary flex items-center gap-2"
          >
            <RotateCcw size={16} />
            Try Again
          </button>
          <details className="mt-4 text-xs text-gray-700 max-w-md">
            <summary className="cursor-pointer hover:text-gray-500">
              Error details
            </summary>
            <pre className="mt-2 p-3 bg-gray-900 rounded-lg overflow-x-auto text-left">
              {this.state.error?.stack}
            </pre>
          </details>
        </div>
      );
    }

    return this.props.children;
  }
}

// Skeleton loaders for pages while data is loading.
export function PageSkeleton() {
  return (
    <div className="p-6 space-y-6 animate-pulse">
      {/* Header skeleton */}
      <div className="flex items-center justify-between">
        <div>
          <div className="h-7 w-48 bg-gray-800 rounded-lg" />
          <div className="h-4 w-72 bg-gray-800/60 rounded mt-2" />
        </div>
        <div className="h-9 w-24 bg-gray-800 rounded-lg" />
      </div>

      {/* Cards skeleton */}
      <div className="grid grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="card h-24">
            <div className="h-8 w-16 bg-gray-800 rounded mb-2" />
            <div className="h-3 w-20 bg-gray-800/60 rounded" />
          </div>
        ))}
      </div>

      {/* Content skeleton */}
      <div className="card space-y-3">
        <div className="h-5 w-40 bg-gray-800 rounded" />
        <div className="h-4 w-full bg-gray-800/40 rounded" />
        <div className="h-4 w-3/4 bg-gray-800/40 rounded" />
        <div className="h-4 w-5/6 bg-gray-800/40 rounded" />
      </div>

      <div className="card h-48">
        <div className="h-full w-full bg-gray-800/30 rounded-lg" />
      </div>
    </div>
  );
}

// CardSkeleton for individual card loading states.
export function CardSkeleton({ height = "h-24" }: { height?: string }) {
  return (
    <div className={`card ${height} animate-pulse`}>
      <div className="h-full flex items-center gap-3">
        <div className="w-10 h-10 bg-gray-800 rounded-lg shrink-0" />
        <div className="flex-1 space-y-2">
          <div className="h-4 w-24 bg-gray-800 rounded" />
          <div className="h-3 w-16 bg-gray-800/60 rounded" />
        </div>
      </div>
    </div>
  );
}

// TableSkeleton for list/table loading states.
export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="card p-0 overflow-hidden animate-pulse">
      <div className="h-10 bg-gray-800/30 border-b border-gray-800" />
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          className="h-12 border-b border-gray-800/50 flex items-center px-4 gap-4"
        >
          <div className="w-3 h-3 bg-gray-800 rounded-full" />
          <div className="h-3 w-24 bg-gray-800/50 rounded" />
          <div className="h-3 w-16 bg-gray-800/40 rounded" />
          <div className="flex-1" />
          <div className="h-3 w-12 bg-gray-800/40 rounded" />
        </div>
      ))}
    </div>
  );
}
