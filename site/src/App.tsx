import { useRef, useMemo, useState } from "react";
import "./App.css";
import * as THREE from "three";
import { useTexture, PerspectiveCamera } from "@react-three/drei";
import { Canvas, useFrame } from "@react-three/fiber";
import { Physics, RapierRigidBody, RigidBody } from "@react-three/rapier";
import { Letter, PButton } from "./models";
import { Text } from "@react-three/drei";

const glassMaterial = new THREE.MeshPhysicalMaterial({
	color: 0xffffff,
	transparent: true,
	opacity: 0.3,

	roughness: 0.4,
	metalness: 0.0,

	transmission: 0.9,
	ior: 1.5,
	thickness: 2.5,

	clearcoat: 0.75,
	clearcoatRoughness: 0.25,
});

function BackgroundImage() {
	const texture = useTexture("/background.png");
	texture.colorSpace = THREE.SRGBColorSpace;

	return <primitive attach="background" object={texture} />;
}

function GradientFloorMaterial() {
	const material = useMemo(
		() =>
			new THREE.ShaderMaterial({
				toneMapped: false,
				uniforms: {
					color1: { value: new THREE.Color("#ED7C59") },
					color2: { value: new THREE.Color("#ECC46E") },
				},
				vertexShader: `
      varying vec4 vScreenPos;
      void main() {
        vec4 worldPosition = modelMatrix * vec4(position, 1.0);
        gl_Position = projectionMatrix * viewMatrix * worldPosition;
        vScreenPos = gl_Position;
      }
    `,
				fragmentShader: `
      uniform vec3 color1;
      uniform vec3 color2;
      varying vec4 vScreenPos;
      void main() {
        vec2 ndc = (vScreenPos.xy / vScreenPos.w) * 0.5 + 0.5;
        float t = clamp(ndc.y, 0.0, 1.0);
        vec3 color = mix(color2, color1, t);
        gl_FragColor = vec4(color, 1.0); 
      }
    `,
			}),
		[],
	);

	return <primitive object={material} />;
}

function Button({ text, pos, onClick }: any) {
	const [isPressed, setIsPressed] = useState(false);
	const ref = useRef<RapierRigidBody | null>(null);
	const currentY = useRef(5);
	const [position, setPosition] = useState([0, 10, 0]);
	const [rotation, setRotation] = useState([0, 1.5, 2]);
	const speed = 12;

	useFrame((_, delta) => {
		const body = ref.current;
		if (!body) return;

		const target = isPressed ? -1.1664540767669678 - 0.2 : -1.1664540767669678;

		currentY.current = THREE.MathUtils.lerp(
			currentY.current,
			target,
			1 - Math.exp(-speed * delta),
		);

		let currentX = ref.current?.translation().x ?? 0;
		let currentZ = ref.current?.translation().z ?? 0;

		setPosition([currentX, currentY.current, currentZ]);
		setRotation([
			ref.current?.rotation().x ?? 0,
			ref.current?.rotation().y ?? 0,
			ref.current?.rotation().z ?? 0,
		]);

		body.setNextKinematicTranslation({
			x: currentX,
			y: currentY.current,
			z: currentZ,
		});
	});

	return (
		<>
			<RigidBody
				ref={ref}
				type={"dynamic"}
				colliders="cuboid"
				position={pos}
				rotation={[0, 1.5, 2]}
				scale={[0.5, 0.5, 0.5]}
				mass={100}
			>
				<group>
					<PButton
						scale={[1, 0.8, 0.8]}
						onClick={onClick}
						onPointerDown={() => {
							setIsPressed(true);
							ref.current?.setBodyType(2, true); // kinematicPosition
						}}
						onPointerUp={() => {
							setIsPressed(false);
							ref.current?.setBodyType(0, true); // dynamic
						}}
						onPointerLeave={() => {
							setIsPressed(false);
							ref.current?.setBodyType(0, true); // dynamic
						}}
					/>
					<Text
						fontSize={0.5}
						position={[0.8, 0, 0]}
						rotation={[-1.5, 1, -1.62]}
						font="/space.ttf"
					>
						{text}
					</Text>
				</group>
			</RigidBody>
		</>
	);
}

function FLetter({ index }: any) {
	const word = "pointhole";
	const letter = word[index];
	const dist = 0.5;
	const letterOffset = index * dist - (word.length * dist) / 2;
	const letterFixed = {
		p: 0,
		o: 0.1,
		i: -0.02,
		n: 0.1,
		t: 0.05,
		h: 0.1,
		l: 0.15,
		e: 0.2,
	};
	// @ts-ignore
	const pos = [letterOffset, 4, (letterFixed[letter] || 0) - 3];
	const frontMaterial = new THREE.MeshStandardMaterial({ color: "#7c7c7c" });
	const ref = useRef<RapierRigidBody | null>(null);
	return (
		<RigidBody
			type="dynamic"
			colliders="hull"
			position={[0, -4, 0]}
			scale={3}
			mass={10}
			ref={ref}
		>
			<Letter
				onPointerOver={() => {
					if (ref.current) {
						ref.current.applyImpulse({ x: 0, y: 0.8, z: 0 }, true);
					}
				}}
				onPointerDown={() => {
					if (ref.current) {
						ref.current.applyImpulse({ x: 0, y: 0, z: -0.5 }, true);
					}
				}}
				index={letter}
				position={pos}
				rotation={[0, 0, 0]}
				material={frontMaterial}
			></Letter>
		</RigidBody>
	);
}

function download(s: boolean, os: boolean) {
	const urls = {
		server: {
			linux: "https://cdn.lu2000luk.com/pointhole/server/server",
			windows: "https://cdn.lu2000luk.com/pointhole/server/server.exe",
		},
		client: {
			linux: "https://cdn.lu2000luk.com/pointhole/client/client",
			windows: "https://cdn.lu2000luk.com/pointhole/client/client.exe",
		},
	};
	if (s) {
		if (os) {
			window.open(urls.server.windows + "?date=" + Date.now(), "_blank");
		} else {
			window.open(urls.server.linux + "?date=" + Date.now(), "_blank");
		}
	} else {
		if (os) {
			window.open(urls.client.windows + "?date=" + Date.now(), "_blank");
		} else {
			window.open(urls.client.linux + "?date=" + Date.now(), "_blank");
		}
	}
}

function App() {
	const [scriptsOpen, setScriptsOpen] = useState(false);

	const scripts = {
		client: {
			windows: `powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-client.ps1 | iex"`,
			linux: `curl -fsSL https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-client.sh | bash`,
		},
		server: {
			windows: `powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-server.ps1 | iex"`,
			linux: `curl -fsSL https://raw.githubusercontent.com/lu2000luk/pointhole/master/install-server.sh | bash`,
		},
	};

	const [serverSelected, setServerSelected] = useState(true);
	const [copiedLinux, setCopiedLinux] = useState(false);
	const [copiedWindows, setCopiedWindows] = useState(false);

	return (
		<>
			{scriptsOpen && (
				<>
					<div
						style={{
							height: "100vh",
							width: "100vw",
							position: "absolute",
							top: 0,
							left: 0,
							backgroundColor: "rgba(0, 0, 0, 0.6)",
							backdropFilter: "blur(10px)",
							zIndex: 10,
						}}
					></div>

					<div
						className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-[11] w-[80%] max-w-[600px] rounded-[20px] p-6 shadow-2xl"
						style={{ backgroundColor: "#2a2a2a" }}
					>
						<div className="flex items-center justify-between mb-5">
							<p className="text-lg font-medium text-white">Install scripts</p>

							<div className="flex items-center gap-2">
								<div className="flex gap-2 bg-[#3a3a3a] rounded-full p-1 cursor-pointer">
									<button
										className={`rounded-full px-[18px] py-[6px] text-sm font-medium cursor-pointer ${
											serverSelected
												? "bg-[#4a4a4a] text-[#e0e0e0]"
												: "bg-transparent text-[#a0a0a0]"
										}`}
										onClick={() => setServerSelected(true)}
									>
										Server
									</button>
									<button
										className={`rounded-full px-[18px] py-[6px] text-sm font-medium cursor-pointer ${
											!serverSelected
												? "bg-[#4a4a4a] text-[#e0e0e0]"
												: "bg-transparent text-[#a0a0a0]"
										}`}
										onClick={() => setServerSelected(false)}
									>
										Client
									</button>
								</div>

								<div>
									<button
										className="bg-black/15 text-[#c0c0c0] cursor-pointer rounded-full px-1.5 py-1.5 text-sm font-medium hover:opacity-80 transition-opacity"
										onClick={() => setScriptsOpen(false)}
									>
										<svg
											xmlns="http://www.w3.org/2000/svg"
											width="20"
											height="20"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="2"
											stroke-linecap="round"
											stroke-linejoin="round"
										>
											<path d="M18 6 6 18" />
											<path d="m6 6 12 12" />
										</svg>
									</button>
								</div>
							</div>
						</div>

						{/* Linux */}
						<div className="mb-3">
							<p className="text-sm text-[#b0b0b0] mb-1.5 ml-1">For linux</p>
							<div className="flex items-center gap-2">
								<div className="flex-1 bg-[#1a1a1a] rounded-full px-3.5 py-2.5 flex items-center overflow-x-auto">
									<code className="font-mono text-sm text-green-400 whitespace-nowrap">
										{serverSelected
											? scripts.server.linux
											: scripts.client.linux}
									</code>
								</div>
								<button
									className="bg-[#3a3a3a] text-[#c0c0c0] rounded-full px-4 py-2 text-sm font-medium cursor-pointer hover:opacity-85 transition-opacity"
									onClick={() => {
										const script = serverSelected
											? scripts.server.linux
											: scripts.client.linux;
										navigator.clipboard.writeText(script);
										setCopiedLinux(true);
										setTimeout(() => setCopiedLinux(false), 2000);
									}}
								>
									{copiedLinux ? "Copied!" : "Copy"}
								</button>
							</div>
						</div>

						{/* Windows */}
						<div>
							<p className="text-sm text-[#b0b0b0] mb-1.5 ml-1">For windows</p>
							<div className="flex items-center gap-2">
								<div className="flex-1 bg-[#1a1a1a] rounded-full px-3.5 py-2.5 flex items-center overflow-x-auto">
									<code className="font-mono text-sm text-blue-400 whitespace-nowrap">
										{serverSelected
											? scripts.server.windows
											: scripts.client.windows}
									</code>
								</div>
								<button
									className="bg-[#3a3a3a] text-[#c0c0c0] rounded-full px-4 py-2 text-sm font-medium cursor-pointer hover:opacity-85 transition-opacity"
									onClick={() => {
										const script = serverSelected
											? scripts.server.windows
											: scripts.client.windows;
										navigator.clipboard.writeText(script);
										setCopiedWindows(true);
										setTimeout(() => setCopiedWindows(false), 2000);
									}}
								>
									{copiedWindows ? "Copied!" : "Copy"}
								</button>
							</div>
						</div>
					</div>
				</>
			)}

			<div style={{ height: "100vh", width: "100vw" }}>
				<Canvas>
					{/* <BackgroundImage /> */}

					<Physics>
						<pointLight distance={10} intensity={20} color="yellow" />

						<Button
							text="Windows"
							pos={[-5, 4, 0]}
							onClick={() => download(true, true)}
						/>
						<Button
							text="Linux"
							pos={[-2, 4, 0]}
							onClick={() => download(true, false)}
						/>

						<Button
							text="Windows"
							pos={[2, 4, 0]}
							onClick={() => download(false, true)}
						/>
						<Button
							text="Linux"
							pos={[5, 4, 0]}
							onClick={() => download(false, false)}
						/>

						<RigidBody type="dynamic" position={[0, 1, -8]} colliders="cuboid">
							<mesh material={glassMaterial}>
								<boxGeometry args={[16, 0.5, 3]} />
							</mesh>
						</RigidBody>

						<RigidBody
							type="dynamic"
							position={[3.5, 1, -2.5]}
							colliders="cuboid"
						>
							<group>
								<mesh material={glassMaterial}>
									<boxGeometry args={[2, 0.25, 0.8]} />
								</mesh>

								<Text
									fontSize={0.4}
									rotation={[-1.6, 0, 0]}
									position={[0, 0.2, 0]}
									font="/bitcount.ttf"
								>
									Client
								</Text>
							</group>
						</RigidBody>

						<RigidBody
							type="dynamic"
							position={[-3.5, 1, -2.5]}
							colliders="cuboid"
						>
							<group>
								<mesh material={glassMaterial}>
									<boxGeometry args={[2, 0.25, 0.8]} />
								</mesh>

								<Text
									fontSize={0.4}
									rotation={[-1.6, 0, 0]}
									position={[0, 0.2, 0]}
									font="/bitcount.ttf"
								>
									Server
								</Text>
							</group>
						</RigidBody>

						<FLetter index={0} />
						<FLetter index={1} />
						<FLetter index={2} />
						<FLetter index={3} />
						<FLetter index={4} />
						<FLetter index={5} />
						<FLetter index={6} />
						<FLetter index={7} />
						<FLetter index={8} />

						<Button
							text="Scripts"
							pos={[0, 4, 2]}
							onClick={() => setScriptsOpen(!scriptsOpen)}
						/>

						<RigidBody type="fixed">
							<mesh position={[0, -2, 0]} rotation={[0, 0, 0]}>
								<cylinderGeometry args={[100, 100, 1, 32]} />
								<GradientFloorMaterial />
							</mesh>
						</RigidBody>
					</Physics>

					<PerspectiveCamera
						makeDefault
						position={[0, 16, 5]}
						rotation={[-1, 0, 0]}
					/>

					<ambientLight intensity={0.5} />
					<directionalLight position={[5, 5, 5]} intensity={1} />
				</Canvas>
			</div>
		</>
	);
}

export default App;
