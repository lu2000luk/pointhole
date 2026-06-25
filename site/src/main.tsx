import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import { useGLTF } from '@react-three/drei'
import App from './App.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)

useGLTF.preload('/button/button.glb')

useGLTF.preload('/letters/p.glb')

useGLTF.preload('/letters/o.glb')

useGLTF.preload('/letters/i.glb')

useGLTF.preload('/letters/n.glb')

useGLTF.preload('/letters/t.glb')

useGLTF.preload('/letters/h.glb')

// useGLTF.preload('/letters/o.glb')

useGLTF.preload('/letters/l.glb')

useGLTF.preload('/letters/e.glb')