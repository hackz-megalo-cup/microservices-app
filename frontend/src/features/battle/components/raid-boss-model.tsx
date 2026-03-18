import { OrbitControls, useGLTF } from "@react-three/drei";
import { Canvas, useFrame } from "@react-three/fiber";
import { Suspense, useMemo, useRef } from "react";
import { type Group, Vector3 } from "three";

const MODEL_PATHS = ["/python.glb", "/swift.glb"] as const;
const MODEL_SCALE = 2.5;
const SCALE_DEFAULT = new Vector3(MODEL_SCALE, MODEL_SCALE, MODEL_SCALE);

function pickRandomModel(): string {
  return MODEL_PATHS[Math.floor(Math.random() * MODEL_PATHS.length)];
}

// Preload both models
for (const path of MODEL_PATHS) {
  useGLTF.preload(path);
}

function Model({ url, squashing }: { url: string; squashing: boolean }) {
  const group = useRef<Group>(null);
  const { scene } = useGLTF(url);

  useFrame(() => {
    if (!group.current) {
      return;
    }
    // Squash effect on hit
    if (squashing) {
      group.current.scale.set(MODEL_SCALE * 1.03, MODEL_SCALE * 0.97, MODEL_SCALE * 1.03);
    } else {
      group.current.scale.lerp(SCALE_DEFAULT, 0.2);
    }
  });

  return (
    <group ref={group}>
      <primitive object={scene} />
    </group>
  );
}

export function RaidBossModel({ squashing }: { squashing: boolean }) {
  const modelUrl = useMemo(() => pickRandomModel(), []);

  return (
    <Canvas
      camera={{ position: [0, 1.5, 4], fov: 45, near: 0.1 }}
      style={{ width: "100%", height: "100%", pointerEvents: "none" }}
    >
      <ambientLight intensity={0.8} />
      <directionalLight position={[5, 5, 5]} intensity={1.2} />
      <directionalLight position={[-3, 2, -3]} intensity={0.4} />
      <Suspense fallback={null}>
        <Model url={modelUrl} squashing={squashing} />
      </Suspense>
      <OrbitControls enabled={false} />
    </Canvas>
  );
}
