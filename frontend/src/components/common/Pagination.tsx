interface PaginationProps {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
}

export default function Pagination({ currentPage, totalPages, onPageChange }: PaginationProps) {
  const pages = Array.from({ length: totalPages }, (_, i) => i + 1)
  
  // Show max 7 page numbers
  const getVisiblePages = () => {
    if (totalPages <= 7) return pages
    
    if (currentPage <= 4) {
      return [...pages.slice(0, 5), -1, totalPages]
    }
    
    if (currentPage >= totalPages - 3) {
      return [1, -1, ...pages.slice(totalPages - 5)]
    }
    
    return [1, -1, currentPage - 1, currentPage, currentPage + 1, -1, totalPages]
  }

  const visiblePages = getVisiblePages()

  return (
    <div className="flex items-center justify-center gap-2 mt-6">
      <button
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage === 1}
        className="px-3 py-2 rounded-md bg-background-tertiary text-text-primary disabled:opacity-50 disabled:cursor-not-allowed hover:bg-background-secondary transition-colors"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
        </svg>
      </button>

      {visiblePages.map((page, index) => {
        if (page === -1) {
          return (
            <span key={`ellipsis-${index}`} className="px-3 py-2 text-text-muted">
              ...
            </span>
          )
        }

        return (
          <button
            key={page}
            onClick={() => onPageChange(page)}
            className={`px-4 py-2 rounded-md transition-colors ${
              currentPage === page
                ? 'bg-accent-primary text-white'
                : 'bg-background-tertiary text-text-primary hover:bg-background-secondary'
            }`}
          >
            {page}
          </button>
        )
      })}

      <button
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage === totalPages}
        className="px-3 py-2 rounded-md bg-background-tertiary text-text-primary disabled:opacity-50 disabled:cursor-not-allowed hover:bg-background-secondary transition-colors"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
      </button>
    </div>
  )
}
