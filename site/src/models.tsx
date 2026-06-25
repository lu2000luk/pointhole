// @ts-nocheck

import { useGLTF } from "@react-three/drei";

export function PButton({ ...props }: any) {
	const { nodes, materials } = useGLTF("/button/button.glb");

	return (
		<mesh
			castShadow
			receiveShadow
			geometry={nodes.Cube.geometry}
			material={materials["Material.001"]}
			{...props}
		/>
	);
}

export function Letter({ index, children, ...props }: any) {
	const { nodes } = useGLTF(`/letters/${index}.glb`);

	// Find the first actual mesh node
	const meshNode = Object.values(nodes).find((n: any) => n.type === "Mesh");

	if (!meshNode) {
		console.error(`Letter model for index ${index} not found.`);
		return null;
	}

	return (
		<mesh castShadow receiveShadow geometry={meshNode.geometry} {...props}>
			{children}
		</mesh>
	);
}
