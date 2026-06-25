import { useState } from 'react'
import './App.css'
import * as THREE from 'three'
import { Canvas } from '@react-three/fiber'

const glassMaterial = new THREE.MeshPhysicalMaterial({
  color: 0xffffff,
  transparent: true,
  opacity: 0.3,

  roughness: 0.1,
  metalness: 0.0,

  transmission: 0.9,
  ior: 1.5,
  thickness: 1.0,

  clearcoat: 1.0,
  clearcoatRoughness: 0.0,
});

function App() {
  const [count, setCount] = useState(0)

  return (
    <div style={{ height: '100vh', width: '100vw' }} onClick={() => setCount(count + 0.1)}>
      <Canvas>

        <mesh material={glassMaterial} position={[0, 0, 0]} rotation={[0, count, count]}>
          <boxGeometry args={[1, 1, 1]} />
        </mesh>
      </Canvas>
    </div>
  )
}

export default App
