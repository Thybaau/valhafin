import { useState } from 'react'
import { useLocation } from 'react-router-dom'

interface PageTransitionProps {
  children: React.ReactNode
}

export default function PageTransition({ children }: PageTransitionProps) {
  const location = useLocation()
  const [displayLocation, setDisplayLocation] = useState(location)
  const [transitionStage, setTransitionStage] = useState('fadeIn')

  // Use a ref to track if we need to transition
  if (location.pathname !== displayLocation.pathname && transitionStage === 'fadeIn') {
    setTransitionStage('fadeOut')
  }

  return (
    <div
      className={`${
        transitionStage === 'fadeOut' ? 'animate-fade-out' : 'animate-fade-in'
      }`}
      onAnimationEnd={() => {
        if (transitionStage === 'fadeOut') {
          setTransitionStage('fadeIn')
          setDisplayLocation(location)
        }
      }}
    >
      {children}
    </div>
  )
}
