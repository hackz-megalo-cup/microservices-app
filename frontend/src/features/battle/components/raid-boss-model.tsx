import { OrbitControls, useAnimations, useGLTF } from "@react-three/drei";
import { Canvas, useFrame } from "@react-three/fiber";
import { Suspense, useEffect, useMemo, useRef } from "react";
import { type Group, Vector3 } from "three";

const MODELS = [
  { path: "/python.glb", rotationY: 0, bg: "/doukutu.png", scale: 2.5 },
  { path: "/swift.glb", rotationY: Math.PI / 4 + Math.PI + 0.6, bg: "/sora.png", scale: 3.0 },
] as const;

function pickRandomModel() {
  return MODELS[Math.floor(Math.random() * MODELS.length)];
}

// Preload both models
for (const { path } of MODELS) {
  useGLTF.preload(path);
}

function Model({
  url,
  rotationY,
  scale,
  squashing,
}: {
  url: string;
  rotationY: number;
  scale: number;
  squashing: boolean;
}) {
  const group = useRef<Group>(null);
  const { scene, animations } = useGLTF(url);
  const { actions, names } = useAnimations(animations, group);
  const scaleDefault = useMemo(() => new Vector3(scale, scale, scale), [scale]);

  useEffect(() => {
    for (const name of names) {
      actions[name]?.reset().play();
    }
    return () => {
      for (const name of names) {
        actions[name]?.stop();
      }
    };
  }, [actions, names]);

  useFrame(() => {
    if (!group.current) {
      return;
    }
    // Squash effect on hit
    if (squashing) {
      group.current.scale.set(scale * 1.03, scale * 0.97, scale * 1.03);
    } else {
      group.current.scale.lerp(scaleDefault, 0.2);
    }
  });

  return (
    <group ref={group} rotation={[0, rotationY, 0]}>
      <primitive object={scene} />
    </group>
  );
}

export function useRaidBossModel() {
  return useMemo(() => pickRandomModel(), []);
}

export function RaidBossModel({
  squashing,
  model,
}: {
  squashing: boolean;
  model: (typeof MODELS)[number];
}) {
  return (
    <Canvas
      camera={{ position: [0, 1.5, 4], fov: 45, near: 0.1 }}
      style={{ width: "100%", height: "100%", pointerEvents: "none" }}
    >
      <ambientLight intensity={0.8} />
      <directionalLight position={[5, 5, 5]} intensity={1.2} />
      <directionalLight position={[-3, 2, -3]} intensity={0.4} />
      <Suspense fallback={null}>
        <Model
          url={model.path}
          rotationY={model.rotationY}
          scale={model.scale}
          squashing={squashing}
        />
      </Suspense>
      <OrbitControls enabled={false} />
    </Canvas>
  );
}
