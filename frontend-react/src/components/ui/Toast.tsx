import React, { useEffect, useState } from 'react';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

interface ToastProps {
  message: string;
  type?: ToastType;
  duration?: number;
  onClose?: () => void;
}

const Toast: React.FC<ToastProps> = ({
  message,
  type = 'info',
  duration = 3000,
  onClose,
}) => {
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsVisible(false);
      setTimeout(() => onClose?.(), 300);
    }, duration);

    return () => clearTimeout(timer);
  }, [duration, onClose]);

  const typeStyles = {
    success: 'bg-green-50 border-green-400 text-green-700',
    error: 'bg-red-50 border-red-400 text-red-700',
    warning: 'bg-yellow-50 border-yellow-400 text-yellow-700',
    info: 'bg-blue-50 border-blue-400 text-blue-700',
  };

  const typeIcons = {
    success: '✅',
    error: '❌',
    warning: '⚠️',
    info: 'ℹ️',
  };

  if (!isVisible) return null;

  return (
    <div className={`fixed top-4 right-4 z-50 max-w-md animate-slideInRight ${isVisible ? 'opacity-100' : 'opacity-0'} transition-opacity duration-300`}>
      <div className={`border-l-4 p-4 rounded-lg shadow-lg ${typeStyles[type]}`}>
        <div className="flex items-start gap-3">
          <span className="text-xl">{typeIcons[type]}</span>
          <p className="flex-1 text-sm font-medium">{message}</p>
          <button
            onClick={() => {
              setIsVisible(false);
              setTimeout(() => onClose?.(), 300);
            }}
            className="text-gray-400 hover:text-gray-600"
          >
            ✕
          </button>
        </div>
      </div>
    </div>
  );
};

// Toast container para múltiplos toasts
interface ToastContainerProps {
  toasts: Array<{ id: string; message: string; type?: ToastType; duration?: number }>;
  onRemove: (id: string) => void;
}

export const ToastContainer: React.FC<ToastContainerProps> = ({ toasts, onRemove }) => {
  return (
    <>
      {toasts.map((toast) => (
        <Toast
          key={toast.id}
          message={toast.message}
          type={toast.type}
          duration={toast.duration}
          onClose={() => onRemove(toast.id)}
        />
      ))}
    </>
  );
};

export default Toast;