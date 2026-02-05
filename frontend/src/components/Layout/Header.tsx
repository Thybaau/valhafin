interface HeaderProps {
  title: string
  subtitle?: string
  actions?: React.ReactNode
}

export default function Header({ title, subtitle, actions }: HeaderProps) {
  return (
    <header className="bg-background-secondary border-b border-background-tertiary px-4 sm:px-6 lg:px-8 py-4 sm:py-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="pt-12 lg:pt-0">
          <h1 className="text-2xl sm:text-3xl font-bold text-text-primary">{title}</h1>
          {subtitle && (
            <p className="text-text-secondary mt-1 text-sm sm:text-base">{subtitle}</p>
          )}
        </div>
        {actions && (
          <div className="flex items-center gap-3 flex-wrap">
            {actions}
          </div>
        )}
      </div>
    </header>
  )
}
