import React from 'react';

interface CardProps {
  children: React.ReactNode;
  title?: string;
  subtitle?: string;
  icon?: React.ReactNode;
  className?: string;
  headerClassName?: string;
  bodyClassName?: string;
  footer?: React.ReactNode;
  hover?: boolean;
  loading?: boolean;
}

const Card: React.FC<CardProps> = ({
  children,
  title,
  subtitle,
  icon,
  className = '',
  headerClassName = '',
  bodyClassName = '',
  footer,
  hover = false,
  loading = false,
}) => {
  return (
    <div className={`bg-white rounded-xl shadow-md overflow-hidden ${hover ? 'hover:shadow-lg transition-shadow duration-200' : ''} ${className}`}>
      {(title || subtitle || icon) && (
        <div className={`px-6 py-4 border-b border-gray-100 flex items-center gap-3 ${headerClassName}`}>
          {icon && <span className="text-2xl">{icon}</span>}
          <div>
            {title && <h3 className="text-lg font-semibold text-gray-800">{title}</h3>}
            {subtitle && <p className="text-sm text-gray-500">{subtitle}</p>}
          </div>
        </div>
      )}
      <div className={`p-6 ${bodyClassName} ${loading ? 'opacity-50 pointer-events-none' : ''}`}>
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin text-3xl">⏳</div>
          </div>
        ) : (
          children
        )}
      </div>
      {footer && (
        <div className="px-6 py-4 bg-gray-50 border-t border-gray-100">
          {footer}
        </div>
      )}
    </div>
  );
};

export default Card;