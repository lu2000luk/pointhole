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

useGLTF.preload('/letters/0.glb') // e
useGLTF.preload('/letters/1.glb') // l
useGLTF.preload('/letters/2.glb') // o
useGLTF.preload('/letters/3.glb') // h
useGLTF.preload('/letters/4.glb') // t
useGLTF.preload('/letters/5.glb') // n
useGLTF.preload('/letters/6.glb') // i
useGLTF.preload('/letters/7.glb') // o
useGLTF.preload('/letters/8.glb') // P
