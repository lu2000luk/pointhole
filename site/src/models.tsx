// @ts-nocheck

import { useGLTF } from "@react-three/drei"

export function PButton({ ...props }: any) {
    const { nodes, materials } = useGLTF('/button/button.glb')

    return (
        <mesh
            castShadow
            receiveShadow
            geometry={nodes.Cube.geometry}
            material={materials['Material.001']}
            {...props}
        />
    )
}

export function Letter({ index, ...props }: any) {
    const { nodes, materials } = useGLTF(`/letters/${index}.glb`)
    return (
        <group {...props} dispose={null}>
            <mesh
                castShadow
                receiveShadow
                geometry={nodes.Text.geometry}
                {...props}
            />
        </group>
    )
}